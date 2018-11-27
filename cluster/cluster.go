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

package cluster

import (
	"context"

	"github.com/topfreegames/pitaya/conn/message"
	"github.com/topfreegames/pitaya/constants"
	pcontext "github.com/topfreegames/pitaya/context"
	"github.com/topfreegames/pitaya/interfaces"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/tracing"
)

// RPCServer interface
type RPCServer interface {
	SetPitayaServer(protos.PitayaServer)
	interfaces.Module
}

// RPCClient interface
type RPCClient interface {
	Send(route string, data []byte) error
	SendPush(userID string, frontendSv *Server, push *protos.Push) error
	SendKick(userID string, serverType string, kick *protos.KickMsg) error
	BroadcastSessionBind(uid string) error
	Call(ctx context.Context, rpcType protos.RPCType, route *route.Route, session *session.Session, msg *message.Message, server *Server) (*protos.Response, error)
	interfaces.Module
}

// SDListener interface
type SDListener interface {
	AddServer(*Server)
	RemoveServer(*Server)
}

// RemoteBindingListener listens to session bindings in remote servers
type RemoteBindingListener interface {
	OnUserBind(uid, fid string)
}

// InfoRetriever gets cluster info
// It can be implemented, for exemple, by reading
// env var, config or by accessing the cluster API
type InfoRetriever interface {
	Region() string
}

// Action type for enum
type Action int

// Action values
const (
	ADD Action = iota
	DEL
)

func buildRequest(
	ctx context.Context,
	rpcType protos.RPCType,
	route *route.Route,
	session *session.Session,
	msg *message.Message,
	thisServer *Server,
) (protos.Request, error) {
	req := protos.Request{
		Type: rpcType,
		Msg: &protos.Msg{
			Route: route.String(),
			Data:  msg.Data,
		},
	}
	ctx, err := tracing.InjectSpan(ctx)
	if err != nil {
		logger.Log.Errorf("failed to inject span: %s", err)
	}
	ctx = pcontext.AddToPropagateCtx(ctx, constants.PeerIDKey, thisServer.ID)
	ctx = pcontext.AddToPropagateCtx(ctx, constants.PeerServiceKey, thisServer.Type)
	req.Metadata, err = pcontext.Encode(ctx)
	if err != nil {
		return req, err
	}
	if thisServer.Frontend {
		req.FrontendID = thisServer.ID
	}

	switch msg.Type {
	case message.Request:
		req.Msg.Type = protos.MsgType_MsgRequest
	case message.Notify:
		req.Msg.Type = protos.MsgType_MsgNotify
	}

	if rpcType == protos.RPCType_Sys {
		mid := uint(0)
		if msg.Type == message.Request {
			mid = msg.ID
		}
		req.Msg.Id = uint64(mid)
		req.Session = &protos.Session{
			Id:   session.ID(),
			Uid:  session.UID(),
			Data: session.GetDataEncoded(),
		}
	}

	return req, nil
}
