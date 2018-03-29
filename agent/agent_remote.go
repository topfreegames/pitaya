// Copyright (c) nano Author and TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package agent

import (
	"net"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/internal/packet"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/serialize"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/util"
)

// Remote corresponding to another server
type Remote struct {
	Session          *session.Session         // session
	Srv              reflect.Value            // cached session reflect.Value, this avoids repeated calls to reflect.value(a.Session)
	chDie            chan struct{}            // wait for close
	encoder          codec.PacketEncoder      // binary encoder
	frontendID       string                   // the frontend that sent the request
	reply            string                   // nats reply topic
	rpcClient        cluster.RPCClient        // rpc client
	serializer       serialize.Serializer     // message serializer
	serviceDiscovery cluster.ServiceDiscovery // service discovery
}

// NewRemote create new Remote instance
func NewRemote(
	sess *protos.Session,
	reply string,
	rpcClient cluster.RPCClient,
	encoder codec.PacketEncoder,
	serializer serialize.Serializer,
	serviceDiscovery cluster.ServiceDiscovery,
	frontendID string,
) (*Remote, error) {
	a := &Remote{
		chDie:            make(chan struct{}),
		reply:            reply, // TODO this is totally coupled with NATS
		serializer:       serializer,
		encoder:          encoder,
		rpcClient:        rpcClient,
		serviceDiscovery: serviceDiscovery,
		frontendID:       frontendID,
	}

	// binding session
	s := session.New(a, false)
	s.SetFrontendData(frontendID, sess.GetID())
	s.SetUID(sess.GetUid())
	err := s.SetDataEncoded(sess.GetData())
	if err != nil {
		return nil, err
	}
	a.Session = s
	a.Srv = reflect.ValueOf(s)

	return a, nil
}

// Push pushes the message to the player
func (a *Remote) Push(route string, v interface{}) error {
	if (reflect.TypeOf(a.rpcClient) == reflect.TypeOf(&cluster.NatsRPCClient{}) &&
		a.Session.UID() == "") {
		return constants.ErrNoUIDBind
	}
	switch d := v.(type) {
	case []byte:
		logger.Log.Debugf("Type=Push, ID=%d, UID=%d, Route=%s, Data=%dbytes",
			a.Session.ID(), a.Session.UID(), route, len(d))
	default:
		logger.Log.Debugf("Type=Push, ID=%d, UID=%d, Route=%s, Data=%+v",
			a.Session.ID(), a.Session.UID(), route, v)
	}

	return a.sendPush(
		pendingMessage{typ: message.Push, route: route, payload: v},
		cluster.GetUserMessagesTopic(a.Session.UID()),
	)
}

// ResponseMID reponds the message with mid to the player
func (a *Remote) ResponseMID(mid uint, v interface{}) error {
	if mid <= 0 {
		return constants.ErrSessionOnNotify
	}

	switch d := v.(type) {
	case []byte:
		logger.Log.Debugf("Type=Response, ID=%d, MID=%d, Data=%dbytes",
			a.Session.ID(), mid, len(d))
	default:
		logger.Log.Infof("Type=Response, ID=%d, MID=%d, Data=%+v",
			a.Session.ID(), mid, v)
	}

	return a.send(pendingMessage{typ: message.Response, mid: mid, payload: v}, a.reply)
}

// Close closes the remote
func (a *Remote) Close() error { return nil }

// RemoteAddr returns the remote address of the player
func (a *Remote) RemoteAddr() net.Addr { return nil }

func (a *Remote) serialize(m pendingMessage) ([]byte, error) {
	payload, err := util.SerializeOrRaw(a.serializer, m.payload)
	if err != nil {
		return nil, err
	}

	// construct message and encode
	msg := &message.Message{
		Type:  m.typ,
		Data:  payload,
		Route: m.route,
		ID:    m.mid,
	}
	em, err := msg.Encode()
	if err != nil {
		return nil, err
	}

	// packet encode
	p, err := a.encoder.Encode(packet.Data, em)
	if err != nil {
		return nil, err
	}

	return p, err
}

func (a *Remote) send(m pendingMessage, to string) (err error) {
	p, err := a.serialize(m)
	if err != nil {
		return err
	}
	res := &protos.Response{
		Data: p,
	}
	bt, err := proto.Marshal(res)
	if err != nil {
		return err
	}
	return a.rpcClient.Send(to, bt)
}

func (a *Remote) sendPush(m pendingMessage, to string) (err error) {
	payload, err := util.SerializeOrRaw(a.serializer, m.payload)
	if err != nil {
		return err
	}
	push := &protos.Push{
		Route: m.route,
		Uid:   a.Session.UID(),
		Data:  payload,
	}
	msg, err := proto.Marshal(push)
	if err != nil {
		return err
	}
	return a.rpcClient.Send(to, msg)
}

// SendRequest sends a request to a server
func (a *Remote) SendRequest(serverID, reqRoute string, v interface{}) (*protos.Response, error) {
	payload, err := util.SerializeOrRaw(a.serializer, v)
	if err != nil {
		return nil, err
	}
	msg := &message.Message{
		Route: reqRoute,
		Data:  payload,
	}
	server, err := a.serviceDiscovery.GetServer(serverID)
	if err != nil {
		return nil, err
	}
	r, err := route.Decode(reqRoute)
	if err != nil {
		return nil, err
	}
	return a.rpcClient.Call(protos.RPCType_User, r, nil, msg, server)
}
