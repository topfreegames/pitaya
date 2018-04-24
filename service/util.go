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
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"reflect"

	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/constants"
	e "github.com/topfreegames/pitaya/errors"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/pipeline"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/serialize"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/util"
)

var errInvalidMsg = errors.New("invalid message type provided")

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
			return nil, err
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
		return msgType, errInvalidMsg
	}
	return msgType, nil
}

func executeBeforePipeline(ctx context.Context, data []byte) ([]byte, error) {
	var err error
	res := data
	if len(pipeline.BeforeHandler.Handlers) > 0 {
		for _, h := range pipeline.BeforeHandler.Handlers {
			res, err = h(ctx, res)
			if err != nil {
				// TODO: not sure if this should be logged
				// one may want to have a before filter that prevents handler execution
				// example: auth
				logger.Log.Errorf("pitaya/handler: broken pipeline: %s", err.Error())
				return res, err
			}
		}
	}
	return res, nil
}

func executeAfterPipeline(ctx context.Context, ser serialize.Serializer, res []byte) []byte {
	var err error
	ret := res
	if len(pipeline.AfterHandler.Handlers) > 0 {
		for _, h := range pipeline.AfterHandler.Handlers {
			ret, err = h(ctx, ret)
			if err != nil {
				logger.Log.Debugf("broken pipeline, error: %s", err.Error())
				// err can be ignored since serializer was already tested previously
				ret, _ = util.GetErrorPayload(ser, err)
				return ret
			}
		}
	}
	return ret
}

func serializeReturn(ser serialize.Serializer, ret interface{}) ([]byte, error) {
	res, err := util.SerializeOrRaw(ser, ret)
	if err != nil {
		logger.Log.Error(err.Error())
		res, err = util.GetErrorPayload(ser, err)
		if err != nil {
			logger.Log.Error("cannot serialize message and respond to the client ", err.Error())
			return nil, err
		}
	}
	return res, nil
}

func processHandlerMessage(
	ctx context.Context,
	rt *route.Route,
	serializer serialize.Serializer,
	session *session.Session,
	data []byte,
	msgTypeIface interface{},
	remote bool,
) ([]byte, error) {
	ctx = context.WithValue(ctx, constants.SessionCtxKey, session)
	h, err := getHandler(rt)
	if err != nil {
		return nil, e.NewError(err, e.ErrNotFoundCode)
	}

	msgType, err := getMsgType(msgTypeIface)
	if err != nil {
		return nil, e.NewError(err, e.ErrInternalCode)
	}
	exit, err := h.ValidateMessageType(msgType)
	if err != nil && exit {
		return nil, e.NewError(err, e.ErrBadRequestCode)
	} else if err != nil {
		logger.Log.Warn(err.Error())
	}

	if data, err = executeBeforePipeline(ctx, data); err != nil {
		return nil, err
	}

	arg, err := unmarshalHandlerArg(h, serializer, data)
	if err != nil {
		return nil, e.NewError(err, e.ErrBadRequestCode)
	}

	logger.Log.Debugf("SID=%d, Data=%s", session.ID(), data)
	args := []reflect.Value{h.Receiver, reflect.ValueOf(ctx)}
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
		// the reason why we don't just Publish is to keep track of failed rpc requests
		// with timeouts, maybe we can improve this flow
		resp = []byte("ack")
	}

	ret, err := serializeReturn(serializer, resp)
	if err == nil {
		if r := executeAfterPipeline(ctx, serializer, ret); r != nil {
			return r, nil
		}
	}

	return ret, nil
}
