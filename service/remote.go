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
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/topfreegames/pitaya/agent"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
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
}

// NewRemoteService creates and return a new RemoteService
func NewRemoteService(
	rpcClient cluster.RPCClient,
	rpcServer cluster.RPCServer,
	sd cluster.ServiceDiscovery,
	encoder codec.PacketEncoder,
	serializer serialize.Serializer,
) *RemoteService {
	return &RemoteService{
		services:         make(map[string]*component.Service),
		rpcClient:        rpcClient,
		rpcServer:        rpcServer,
		encoder:          encoder,
		serviceDiscovery: sd,
		serializer:       serializer,
	}
}

var remotes = make(map[string]*component.Remote) // all remote method

func (r *RemoteService) remoteProcess(a *agent.Agent, route *route.Route, msg *message.Message) {
	var res []byte
	var err error
	if res, err = r.remoteCall(protos.RPCType_Sys, route, a.Session, msg); err != nil {
		log.Errorf(err.Error())
		return
	}
	a.WriteToChWrite(res)
}

// RPC makes rpcs
func (r *RemoteService) RPC(route *route.Route, reply interface{}, args ...interface{}) (interface{}, error) {
	data, err := util.GobEncode(args...)
	fmt.Println("ENCODER", string(data))
	if err != nil {
		fmt.Println("AQUI!")
		return nil, err
	}
	msg := &message.Message{
		Type:  message.Request,
		Route: fmt.Sprintf("%s.%s", route.Service, route.Method),
		Data:  data,
	}

	ret, err := r.remoteCall(protos.RPCType_User, route, nil, msg)
	if err != nil {
		return nil, err
	}

	err = util.GobDecode(reply, ret)
	if err != nil {
		return nil, err
	}
	return reply, nil
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
		s := session.GetSessionByUID(push.GetUid())
		fmt.Printf("Got PUSH message %v %v\n", push, s == nil)
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
		switch {
		case req.Type == protos.RPCType_Sys:
			a := agent.NewRemote(req.GetSession(),
				req.GetMsg().GetReply(),
				r.rpcClient,
				r.encoder,
				r.serializer,
			)
			// TODO change requestID name
			a.SetMID(uint(req.GetMsg().GetID()))
			rt, err := route.Decode(req.GetMsg().GetRoute())
			if err != nil {
				// TODO answer rpc with an error
				continue
			}
			h, ok := handlers[fmt.Sprintf("%s.%s", rt.Service, rt.Method)]
			if !ok {
				log.Warnf("pitaya/handler: %s not found", req.GetMsg().GetRoute())
				// TODO answer rpc with an error
				continue
			}
			var data interface{}
			if h.IsRawArg {
				data = req.GetMsg().GetData()
			} else {
				data = reflect.New(h.Type.Elem()).Interface()
				err := r.serializer.Unmarshal(req.GetMsg().GetData(), data)
				if err != nil {
					// TODO answer with error
					logger.Log.Error("deserialize error", err.Error())
					return
				}
			}

			log.Debugf("SID=%d, Data=%s", req.GetSession().GetID(), data)
			// backend session

			// need to create agent
			//handler.processMessage()
			// user request proxied from frontend server
			args := []reflect.Value{h.Receiver, reflect.ValueOf(a.Session), reflect.ValueOf(data)}
			util.Pcall(h.Method, args)
		case req.Type == protos.RPCType_User:
			fmt.Printf("CAMILA req %v %v\n", req, req.GetMsg().GetData())
			response := &protos.Response{}
			rt, err := route.Decode(req.GetMsg().GetRoute())
			if err != nil {
				// TODO answer rpc with an error
				continue
			}
			remote, ok := remotes[fmt.Sprintf("%s.%s", rt.Service, rt.Method)]
			if !ok {
				log.Warnf("pitaya/remote: %s not found", req.GetMsg().GetRoute())
				// TODO answer rpc with an error
				// response.Error =
				continue
			}

			var args []interface{}
			gob.NewDecoder(bytes.NewReader(req.GetMsg().GetData())).Decode(&args)

			params := []reflect.Value{remote.Receiver}
			for _, arg := range args {
				params = append(params, reflect.ValueOf(arg))
			}
			fmt.Printf("ARGS %v \n", args)
			fmt.Printf("PARAMS %v \n", params)

			ret, err := util.PcallReturn(remote.Method, params)
			if err != nil {
				// TODO answer rpc with an error
				// response.Error = err.Error()
				continue
			}
			// remote method encountered an error
			if err := ret[1].Interface(); err != nil {
				// TODO answer rpc with an error
				// response.Error = err.(error).Error()
				continue
			}
			buf := bytes.NewBuffer([]byte(nil))
			if err := gob.NewEncoder(buf).Encode(ret[0].Interface()); err != nil {
				// TODO answer rpc with an error
				// response.Error = err.Error()
				continue
			}
			response.Data = buf.Bytes()
			p, err := proto.Marshal(response)
			if err != nil {
				// TODO answer rpc with an error
				continue
			}
			// TODO is this correct?
			r.rpcClient.Send(req.GetMsg().GetReply(), p)
		}
	}
}

func (r *RemoteService) remoteCall(rpcType protos.RPCType, route *route.Route, session *session.Session, msg *message.Message) ([]byte, error) {
	svType := route.SvType
	//TODO this logic should be elsewhere, routing should be changeable
	serversOfType, err := r.serviceDiscovery.GetServersByType(svType)
	if err != nil {
		return nil, err
	}

	s := rand.NewSource(time.Now().Unix())
	rnd := rand.New(s)
	server := serversOfType[rnd.Intn(len(serversOfType))]

	res, err := r.rpcClient.Call(rpcType, route, session, msg, server)
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
