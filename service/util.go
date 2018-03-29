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
	"errors"
	"fmt"
	"reflect"

	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/pipeline"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/serialize"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/util"
)

func getHandler(rt *route.Route) (*component.Handler, error) {
	handler, ok := handlers[rt.Short()]
	if !ok {
		e := fmt.Errorf("pitaya/handler: %s not found", rt.String())
		return nil, e
	}
	return handler, nil

}

func unmarshalHandlerArg(handler *component.Handler, serializer serialize.Serializer, payload []byte) (interface{}, error) {
	if handler.IsRawArg {
		return payload, nil
	}

	var arg interface{}
	if handler.Type != nil {
		arg = reflect.New(handler.Type.Elem()).Interface()
		err := serializer.Unmarshal(payload, arg)
		if err != nil {
			e := fmt.Errorf("deserialize error: %s", err.Error())
			return nil, e
		}
	}
	return arg, nil
}

func unmarshalRemoteArg(payload []byte) ([]interface{}, error) {
	args := make([]interface{}, 0)
	err := gob.NewDecoder(bytes.NewReader(payload)).Decode(&args)
	if err != nil {
		return nil, err
	}
	return args, nil
}

func getMsgType(msgTypeIface interface{}) (message.Type, error) {
	var msgType message.Type
	if val, ok := msgTypeIface.(message.Type); ok {
		msgType = val
	} else if val, ok := msgTypeIface.(protos.MsgType); ok {
		msgType = util.ConvertProtoToMessageType(val)
	} else {
		return msgType, errors.New("invalid message type provided")
	}
	return msgType, nil
}

func executeBeforePipeline(h *component.Handler, s *session.Session, data []byte) ([]byte, error) {
	if len(pipeline.BeforeHandler.Handlers) > 0 {
		for _, h := range pipeline.BeforeHandler.Handlers {
			res, err := h(s, data)
			if err != nil {
				// TODO: not sure if this should be logged
				// one may want to have a before filter that prevents handler execution
				// example: auth
				log.Errorf("pitaya/handler: broken pipeline: %s", err.Error())
				return res, err
			}
			return res, nil
		}
	}
	return data, nil
}

func executeAfterPipeline(h *component.Handler, s *session.Session, ser serialize.Serializer, res []byte) []byte {
	if len(pipeline.AfterHandler.Handlers) > 0 {
		for _, h := range pipeline.AfterHandler.Handlers {
			ret, err := h(s, res)
			if err != nil {
				log.Debugf("broken pipeline, error: %s", err.Error())
				// err can be ignored since serializer was already tested previously
				ret, _ = util.GetErrorPayload(ser, err)
				return ret
			}
		}
	}
	return nil
}

func serializeReturn(ser serialize.Serializer, ret interface{}) ([]byte, error) {
	res, err := util.SerializeOrRaw(ser, ret)
	if err != nil {
		log.Error(err.Error())
		res, err = util.GetErrorPayload(ser, err)
		if err != nil {
			log.Error("cannot serialize message and respond to the client ", err.Error())
			return nil, err
		}
		return res, err
	}
	return res, nil
}

// TODO: should this be here in utils?
func processHandlerMessage(
	rt *route.Route,
	serializer serialize.Serializer,
	cachedSession reflect.Value,
	session *session.Session,
	data []byte,
	msgTypeIface interface{},
	remote bool,
) ([]byte, error) {
	h, err := getHandler(rt)
	if err != nil {
		return nil, err
	}

	msgType, err := getMsgType(msgTypeIface)
	if err != nil {
		return nil, err
	}
	exit, err := h.ValidateMessageType(msgType)
	if err != nil && exit {
		return nil, err
	} else if err != nil {
		log.Warn(err.Error())
	}

	if data, err = executeBeforePipeline(h, session, data); err != nil {
		return nil, err
	}

	arg, err := unmarshalHandlerArg(h, serializer, data)
	if err != nil {
		return nil, err
	}

	// TODO: find out how to log this without needing the full request
	// log.Debugf("SID=%d, Data=%s", req.GetSession().GetID(), arg)
	args := []reflect.Value{h.Receiver, cachedSession}
	if arg != nil {
		args = append(args, reflect.ValueOf(arg))
	}
	resp, err := util.Pcall(h.Method, args)
	if err != nil {
		return nil, err
	}

	if remote && msgType == message.Notify {
		// This is a special case and should only happen with nats rpc client
		// because we used nats request we have to answer to it or else a timeout
		// will happen in the caller server and will be returned to the client
		// the reason why we not just Publish is to keep track of failed rpc requests
		// with timeouts, maybe we can improve this flow
		resp = []byte("ack")
	}

	ret, err := serializeReturn(serializer, resp)
	if err == nil {
		if r := executeAfterPipeline(h, session, serializer, ret); r != nil {
			return r, nil
		}
	}

	return ret, nil
}
