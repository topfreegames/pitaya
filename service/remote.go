// Copyright (c) TFG Co. All Rights Reserved.
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

package service

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/gogo/protobuf/proto"

	"github.com/topfreegames/pitaya/agent"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/router"
	"github.com/topfreegames/pitaya/serialize"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/util"
)

// RemoteService struct
type RemoteService struct {
	rpcServer        cluster.RPCServer
	serviceDiscovery cluster.ServiceDiscovery
	serializer       serialize.Serializer
	encoder          codec.PacketEncoder
	rpcClient        cluster.RPCClient
	services         map[string]*component.Service // all registered service
	router           *router.Router
}

// NewRemoteService creates and return a new RemoteService
func NewRemoteService(
	rpcClient cluster.RPCClient,
	rpcServer cluster.RPCServer,
	sd cluster.ServiceDiscovery,
	encoder codec.PacketEncoder,
	serializer serialize.Serializer,
	router *router.Router,
) *RemoteService {
	return &RemoteService{
		services:         make(map[string]*component.Service),
		rpcClient:        rpcClient,
		rpcServer:        rpcServer,
		encoder:          encoder,
		serviceDiscovery: sd,
		serializer:       serializer,
		router:           router,
	}
}

var remotes = make(map[string]*component.Remote) // all remote method

func (r *RemoteService) remoteProcess(server *cluster.Server, a *agent.Agent, route *route.Route, msg *message.Message) {
	var res *protos.Response
	var err error
	if res, err = r.remoteCall(server, protos.RPCType_Sys, route, a.Session, msg); err != nil {
		log.Errorf(err.Error())
		return
	}
	// TODO we should not return a response to a notify to the client
	// this is becase of nats
	// we should not send an error to the client because the only thing
	// returned to the client should be the return type in the handler
	if msg.Type == message.Request && res.Error == "" {
		a.Session.ResponseMID(msg.ID, res.Data)
	} else {
		log.Error(res.Error)
	}
}

// RPC makes rpcs
func (r *RemoteService) RPC(serverID string, route *route.Route, reply interface{}, args ...interface{}) error {
	data, err := util.GobEncode(args...)
	if err != nil {
		return err
	}
	msg := &message.Message{
		Type:  message.Request,
		Route: fmt.Sprintf("%s.%s", route.Service, route.Method),
		Data:  data,
	}

	target, _ := r.serviceDiscovery.GetServer(serverID)
	if serverID != "" && target == nil {
		return constants.ErrServerNotFound
	}

	res, err := r.remoteCall(target, protos.RPCType_User, route, nil, msg)
	if err != nil {
		return err
	}

	if res.Error != "" {
		return errors.New(res.Error)
	}

	err = util.GobDecode(reply, res.GetData())
	if err != nil {
		return err
	}
	return nil
}

// Register registers components
func (r *RemoteService) Register(comp component.Component, opts []component.Option) error {
	s := component.NewService(comp, opts)

	if _, ok := r.services[s.Name]; ok {
		return fmt.Errorf("remote: service already defined: %s", s.Name)
	}

	if err := s.ExtractRemote(); err != nil {
		return err
	}

	r.services[s.Name] = s
	// register all remotes
	for name, remote := range s.Remotes {
		remotes[fmt.Sprintf("%s.%s", s.Name, name)] = remote
	}

	return nil
}

// ProcessUserPush receives and processes push to users
// TODO: probably handle concurrency (threadID?)
func (r *RemoteService) ProcessUserPush() {
	for push := range r.rpcServer.GetUserPushChannel() {
		log.Debugf("sending push to user %s: %v", push.GetUid(), string(push.Data))
		s := session.GetSessionByUID(push.GetUid())
		if s != nil {
			s.Push(push.Route, push.Data)
		}
	}
}

// ProcessRemoteMessages processes remote messages
func (r *RemoteService) ProcessRemoteMessages(threadID int) {
	// TODO need to monitor stuff here to guarantee messages are not being dropped
	for req := range r.rpcServer.GetUnhandledRequestsChannel() {
		// TODO should deserializer be decoupled?
		log.Debugf("(%d) processing message %v", threadID, req.GetMsg().GetID())
		rt, err := route.Decode(req.GetMsg().GetRoute())
		if err != nil {
			response := &protos.Response{
				Error: fmt.Sprintf("pitaya: cannot decode route %s", req.GetMsg().GetRoute()),
			}
			r.sendReply(req.GetMsg().GetReply(), response)
			continue
		}

		switch {
		case req.Type == protos.RPCType_Sys:
			r.handleRPCSys(req, rt)
		case req.Type == protos.RPCType_User:
			r.handleRPCUser(req, rt)
		}
	}
}

func (r *RemoteService) handleRPCUser(req *protos.Request, rt *route.Route) {
	reply := req.GetMsg().GetReply()
	response := &protos.Response{}

	remote, ok := remotes[rt.Short()]
	if !ok {
		errMsg := fmt.Sprintf("pitaya/remote: %s not found", rt.Short())
		log.Warnf(errMsg)
		response.Error = errMsg
		r.sendReply(reply, response)
		return
	}

	args, err := unmarshalRemoteArg(req.GetMsg().GetData())
	if err != nil {
		response.Error = err.Error()
		r.sendReply(reply, response)
		return
	}

	params := []reflect.Value{remote.Receiver}
	for _, arg := range args {
		params = append(params, reflect.ValueOf(arg))
	}

	ret, err := util.Pcall(remote.Method, params)
	if err != nil {
		response.Error = err.Error()
		r.sendReply(reply, response)
		return
	}

	res, err := util.GobEncodeSingle(ret)
	if err != nil {
		response.Error = err.Error()
		r.sendReply(reply, response)
		return
	}

	response.Data = res
	r.sendReply(reply, response)
}

func (r *RemoteService) handleRPCSys(req *protos.Request, rt *route.Route) {
	reply := req.GetMsg().GetReply()
	response := &protos.Response{}

	// TODO should we create a new agent for every new request?
	a, err := agent.NewRemote(
		req.GetSession(),
		reply,
		r.rpcClient,
		r.encoder,
		r.serializer,
		r.serviceDiscovery,
		req.FrontendID,
	)
	if err != nil {
		log.Warn("pitaya/handler: cannot instantiate remote agent")
		response.Error = err.Error()
		r.sendReply(reply, response)
		return
	}

	ret, err := processHandlerMessage(rt, r.serializer, a.Srv, a.Session, req.GetMsg().GetData(), req.GetMsg().GetType(), true)
	if err != nil {
		log.Warnf(err.Error())
		response = &protos.Response{
			Error: err.Error(),
		}
	} else {
		response = &protos.Response{
			Data: ret,
		}
	}
	r.sendReply(reply, response)
}

func (r *RemoteService) sendReply(reply string, response *protos.Response) {
	p, err := proto.Marshal(response)
	if err != nil {
		res := &protos.Response{}
		res.Error = err.Error()
		p, _ = proto.Marshal(response)
	}
	r.rpcClient.Send(reply, p)
}

func (r *RemoteService) remoteCall(
	server *cluster.Server,
	rpcType protos.RPCType,
	route *route.Route,
	session *session.Session,
	msg *message.Message,
) (*protos.Response, error) {
	svType := route.SvType

	var err error
	target := server

	if target == nil {
		target, err = r.router.Route(rpcType, svType, session, route)
		if err != nil {
			return nil, err
		}
	}

	res, err := r.rpcClient.Call(rpcType, route, session, msg, target)
	if err != nil {
		return nil, err
	}
	return res, err
}

// DumpServices outputs all registered services
func (r *RemoteService) DumpServices() {
	for name := range remotes {
		log.Infof("registered remote %s", name)
	}
}
