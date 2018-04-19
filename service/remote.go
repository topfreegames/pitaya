//
// Copyright (c) TFG Co. All Rights Reserved.
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
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"

	"github.com/gogo/protobuf/proto"

	"github.com/topfreegames/pitaya/agent"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/constants"
	e "github.com/topfreegames/pitaya/errors"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/logger"
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
		logger.Log.Errorf(err.Error())
		a.AnswerWithError(msg.ID, err)
		return
	}
	if msg.Type == message.Request {
		err := a.Session.ResponseMID(msg.ID, res.Data)
		if err != nil {
			logger.Log.Error(err)
			a.AnswerWithError(msg.ID, err)
		}
	} else if res.Error != nil {
		logger.Log.Errorf("error while sending a notify: %s", res.Error.GetMsg())
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
		Route: route.Short(),
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

	if res.Error != nil {
		return &e.Error{
			Code:     res.Error.Code,
			Message:  res.Error.Msg,
			Metadata: res.Error.Metadata,
		}
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
func (r *RemoteService) ProcessUserPush() {
	for push := range r.rpcServer.GetUserPushChannel() {
		logger.Log.Debugf("sending push to user %s: %v", push.GetUid(), string(push.Data))
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
		logger.Log.Debugf("(%d) processing message %v", threadID, req.GetMsg().GetID())
		rt, err := route.Decode(req.GetMsg().GetRoute())
		if err != nil {
			response := &protos.Response{
				Error: &protos.Error{
					Code: e.ErrBadRequestCode,
					Msg:  "cannot decode route",
					Metadata: map[string]string{
						"route": req.GetMsg().GetRoute(),
					},
				},
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
		logger.Log.Warnf("pitaya/remote: %s not found", rt.Short())
		response := &protos.Response{
			Error: &protos.Error{
				Code: e.ErrNotFoundCode,
				Msg:  "route not found",
				Metadata: map[string]string{
					"route": rt.Short(),
				},
			},
		}
		r.sendReply(reply, response)
		return
	}
	params := []reflect.Value{remote.Receiver}
	if remote.HasArgs {
		args, err := unmarshalRemoteArg(req.GetMsg().GetData())
		if err != nil {
			response := &protos.Response{
				Error: &protos.Error{
					Code: e.ErrBadRequestCode,
					Msg:  err.Error(),
				},
			}
			r.sendReply(reply, response)
			return
		}
		for _, arg := range args {
			params = append(params, reflect.ValueOf(arg))
		}
	}

	ret, err := util.Pcall(remote.Method, params)
	if err != nil {
		response := &protos.Response{
			Error: &protos.Error{
				Code: e.ErrUnknownCode,
				Msg:  err.Error(),
			},
		}
		if val, ok := err.(*e.Error); ok {
			response.Error.Code = val.Code
			if val.Metadata != nil {
				response.Error.Metadata = val.Metadata
			}
		}
		r.sendReply(reply, response)
		return
	}

	buf := bytes.NewBuffer([]byte(nil))
	if ret != nil {
		if err := gob.NewEncoder(buf).Encode(ret); err != nil {
			response := &protos.Response{
				Error: &protos.Error{
					Code: e.ErrUnknownCode,
					Msg:  err.Error(),
				},
			}
			r.sendReply(reply, response)
			return
		}
	}

	response.Data = buf.Bytes()
	r.sendReply(reply, response)
}

func (r *RemoteService) handleRPCSys(req *protos.Request, rt *route.Route) {
	reply := req.GetMsg().GetReply()
	response := &protos.Response{}

	// (warning) a new agent is created for every new request
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
		logger.Log.Warn("pitaya/handler: cannot instantiate remote agent")
		response := &protos.Response{
			Error: &protos.Error{
				Code: e.ErrInternalCode,
				Msg:  err.Error(),
			},
		}
		r.sendReply(reply, response)
		return
	}

	ret, err := processHandlerMessage(rt, r.serializer, a.Srv, a.Session, req.GetMsg().GetData(), req.GetMsg().GetType(), true)
	if err != nil {
		logger.Log.Warnf(err.Error())
		response = &protos.Response{
			Error: &protos.Error{
				Code: e.ErrUnknownCode,
				Msg:  err.Error(),
			},
		}
		if val, ok := err.(*e.Error); ok {
			response.Error.Code = val.Code
			if val.Metadata != nil {
				response.Error.Metadata = val.Metadata
			}
		}
	} else {
		response = &protos.Response{Data: ret}
	}
	r.sendReply(reply, response)
}

func (r *RemoteService) sendReply(reply string, response *protos.Response) {
	p, err := proto.Marshal(response)
	if err != nil {
		response := &protos.Response{
			Error: &protos.Error{
				Code: e.ErrUnknownCode,
				Msg:  err.Error(),
			},
		}
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
			return nil, e.NewError(err, e.ErrInternalCode)
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
		logger.Log.Infof("registered remote %s", name)
	}
}
