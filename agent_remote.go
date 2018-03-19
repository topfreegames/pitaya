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

package pitaya

import (
	"net"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/internal/packet"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/session"
)

// Agent corresponding to another server
type agentRemote struct {
	session *session.Session // session
	lastMid uint             // last message id
	chDie   chan struct{}    // wait for close
	reply   string           // nats reply topic
	srv     reflect.Value    // cached session reflect.Value
}

// Create new agentRemote instance
func newAgentRemote(sess *protos.Session, reply string) *agentRemote {
	a := &agentRemote{
		chDie: make(chan struct{}),
		reply: reply, // TODO this is ugly
	}

	// binding session
	s := session.New(a)
	s.SetUID(sess.GetUid())
	a.session = s

	return a
}

func (a *agentRemote) Push(route string, v interface{}) error {
	switch d := v.(type) {
	case []byte:
		logger.Log.Debugf("Type=Push, ID=%d, UID=%d, Route=%s, Data=%dbytes",
			a.session.ID(), a.session.UID(), route, len(d))
	default:
		logger.Log.Debugf("Type=Push, ID=%d, UID=%d, Route=%s, Data=%+v",
			a.session.ID(), a.session.UID(), route, v)
	}

	return a.sendPush(
		pendingMessage{typ: message.Push, route: route, payload: v},
		cluster.GetUserMessagesTopic(a.session.UID()),
	)
}

func (a *agentRemote) MID() uint { return uint(0) }

func (a *agentRemote) Response(v interface{}) error {
	return a.ResponseMID(a.lastMid, v)
}
func (a *agentRemote) ResponseMID(mid uint, v interface{}) error {
	if mid <= 0 {
		return ErrSessionOnNotify
	}

	switch d := v.(type) {
	case []byte:
		logger.Log.Debugf("Type=Response, ID=%d, MID=%d, Data=%dbytes",
			a.session.ID(), mid, len(d))
	default:
		logger.Log.Infof("Type=Response, ID=%d, MID=%d, Data=%+v",
			a.session.ID(), mid, v)
	}

	return a.send(pendingMessage{typ: message.Response, mid: mid, payload: v}, a.reply)
}

func (a *agentRemote) Close() error         { return nil }
func (a *agentRemote) RemoteAddr() net.Addr { return nil }

func (a *agentRemote) serialize(m pendingMessage) ([]byte, error) {
	payload, err := serializeOrRaw(m.payload)
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
	p, err := app.packetEncoder.Encode(packet.Data, em)
	if err != nil {
		return nil, err
	}

	return p, err
}

func (a *agentRemote) send(m pendingMessage, to string) (err error) {
	p, err := a.serialize(m)
	if err != nil {
		return err
	}
	return app.rpcClient.Answer(to, p)
}

func (a *agentRemote) sendPush(m pendingMessage, to string) (err error) {
	payload, err := serializeOrRaw(m.payload)
	if err != nil {
		return err
	}
	push := &protos.Push{
		Route: m.route,
		Uid:   a.session.UID(),
		Data:  payload,
	}
	msg, err := proto.Marshal(push)
	if err != nil {
		return err
	}
	return app.rpcClient.Answer(to, msg)
}
