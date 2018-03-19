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

package pitaya

import (
	"fmt"
	"reflect"

	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/session"
)

// TODO: probably handle concurrency (threadID?)
func processUserPush() {
	for push := range app.rpcServer.GetUserPushChannel() {
		s := session.GetSessionByUID(push.GetUid())
		fmt.Printf("Got PUSH message %v %v\n", push, s == nil)
		if s != nil {
			s.Push(push.Route, push.Data)
		}
	}
}

func processRemoteMessages(threadID int) {
	// TODO need to monitor stuff here to guarantee messages are not being dropped
	for req := range app.rpcServer.GetUnhandledRequestsChannel() {
		// TODO should deserializer be decoupled?
		log.Debugf("(%d) processing message %v", threadID, req.GetMsg().GetID())
		switch {
		case req.Type == protos.RPCType_Sys:
			agent := newAgentRemote(req.GetSession(), req.GetMsg().GetReply())
			// TODO change requestID name
			agent.lastMid = uint(req.GetMsg().GetID())
			r, err := route.Decode(req.GetMsg().GetRoute())
			if err != nil {
				// TODO answer rpc with an error
				continue
			}
			h, ok := handler.handlers[fmt.Sprintf("%s.%s", r.Service, r.Method)]
			if !ok {
				logger.Log.Warnf("pitaya/handler: %s not found(forgot registered?)", req.GetMsg().GetRoute())
				// TODO answer rpc with an error
				continue
			}
			var data interface{}
			if h.IsRawArg {
				data = req.GetMsg().GetData()
			} else {
				data = reflect.New(h.Type.Elem()).Interface()
				err := app.serializer.Unmarshal(req.GetMsg().GetData(), data)
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
			args := []reflect.Value{h.Receiver, reflect.ValueOf(agent.session), reflect.ValueOf(data)}
			pcall(h.Method, args)
		case req.Type == protos.RPCType_User:
			//TODO
			break
		}
	}

}
