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
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/tutumagi/pitaya/component"
	"github.com/tutumagi/pitaya/conn/message"
	"github.com/tutumagi/pitaya/constants"
	e "github.com/tutumagi/pitaya/errors"
	"github.com/tutumagi/pitaya/logger"
	"github.com/tutumagi/pitaya/pipeline"
	"github.com/tutumagi/pitaya/protos"
	"github.com/tutumagi/pitaya/route"
	"github.com/tutumagi/pitaya/serialize"
	"github.com/tutumagi/pitaya/session"
	"github.com/tutumagi/pitaya/util"
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

func unmarshalRemoteArg(remote *component.Remote, payload []byte) (interface{}, error) {
	var arg interface{}
	if remote.Type != nil {
		arg = reflect.New(remote.Type.Elem()).Interface()
		pb, ok := arg.(proto.Message)
		if !ok {
			return nil, constants.ErrWrongValueType
		}
		err := proto.Unmarshal(payload, pb)
		if err != nil {
			return nil, err
		}
	}
	return arg, nil
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

func executeBeforePipeline(ctx context.Context, data interface{}) (interface{}, error) {
	var err error
	res := data
	if len(pipeline.BeforeHandler.Handlers) > 0 {
		for _, h := range pipeline.BeforeHandler.Handlers {
			res, err = h(ctx, res)
			if err != nil {
				logger.Log.Debugf("pitaya/handler: broken pipeline: %s", err.Error())
				return res, err
			}
		}
	}
	return res, nil
}

func executeRouteBeforePipeline(ctx context.Context, route string, data interface{}) (interface{}, error) {
	var err error
	res := data
	if len(pipeline.BeforeRouterHandler.Handlers[route]) > 0 {
		for _, h := range pipeline.BeforeRouterHandler.Handlers[route] {
			res, err = h(ctx, res)
			if err != nil {
				logger.Log.Debugf("pitaya/handler: before broken route pipeline: %s", err.Error())
				return res, err
			}
		}
	}
	return res, nil
}
func executeAfterPipeline(ctx context.Context, res interface{}, err error) (interface{}, error) {
	ret := res
	if len(pipeline.AfterHandler.Handlers) > 0 {
		for _, h := range pipeline.AfterHandler.Handlers {
			ret, err = h(ctx, ret, err)
		}
	}
	return ret, err
}

func serializeReturn(ser serialize.Serializer, ret interface{}) ([]byte, error) {
	res, err := util.SerializeOrRaw(ser, ret)
	if err != nil {
		logger.Log.Errorf("Failed to serialize return: %s", err.Error())
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
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, constants.SessionCtxKey, session)
	ctx = util.CtxWithDefaultLogger(ctx, rt.String(), session.UID())

	h, err := getHandler(rt)
	if err != nil {
		return nil, e.NewError(err, e.ErrNotFoundCode)
	}

	msgType, err := getMsgType(msgTypeIface)
	if err != nil {
		return nil, e.NewError(err, e.ErrInternalCode)
	}

	logger := ctx.Value(constants.LoggerCtxKey).(logger.Logger)
	exit, err := h.ValidateMessageType(msgType)
	if err != nil && exit {
		return nil, e.NewError(err, e.ErrBadRequestCode)
	} else if err != nil {
		logger.Warnf("invalid message type, router:%s error: %s", rt.Short(), err.Error())
	}

	// First unmarshal the handler arg that will be passed to
	// both handler and pipeline functions
	arg, err := unmarshalHandlerArg(h, serializer, data)
	if err != nil {
		return nil, e.NewError(err, e.ErrBadRequestCode)
	}

	if arg, err = executeBeforePipeline(ctx, arg); err != nil {
		return nil, err
	}

	if arg, err = executeRouteBeforePipeline(ctx, rt.Service, arg); err != nil {
		return nil, err
	}

	// 注释 by 涂飞 日志太多了
	// logger.Debugf("SID=%d, Data=%s", session.ID(), data)
	args := []reflect.Value{h.Receiver, reflect.ValueOf(ctx)}
	if arg != nil {
		args = append(args, reflect.ValueOf(arg))
	}

	resp, err := util.Pcall(h.Method, args)
	// if remote && msgType == message.Notify {
	if msgType == message.Notify { // 不管是不是remote，notify的消息返回值都可能时nil, by tufei
		// This is a special case and should only happen with nats rpc client
		// because we used nats request we have to answer to it or else a timeout
		// will happen in the caller server and will be returned to the client
		// the reason why we don't just Publish is to keep track of failed rpc requests
		// with timeouts, maybe we can improve this flow
		resp = []byte("ack")
	}

	resp, err = executeAfterPipeline(ctx, resp, err)
	if err != nil {
		return nil, err
	}

	ret, err := serializeReturn(serializer, resp)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func directRPCLocalCall(
	ctx context.Context,
	rt *route.Route,
	serializer serialize.Serializer,
	arg interface{},
) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	h, ok := remotes[rt.Short()]
	if !ok {
		logger.Log.Warnf("pitaya/remote local call: %s not found", rt.Short())
		return nil, fmt.Errorf("router not found:%s", rt.Short())
	}

	// h, err := getHandler(rt)
	// if err != nil {
	// 	return nil, e.NewError(err, e.ErrNotFoundCode)
	// }

	args := []reflect.Value{h.Receiver, reflect.ValueOf(ctx)}
	if arg != nil {
		args = append(args, reflect.ValueOf(arg))
	}

	resp, err := util.Pcall(h.Method, args)
	if err != nil {
		return nil, err
	}

	// TODO 这里 返回字节，外面又解码为 结构体，有点重复，编码又立马解码的开销，但是是为了让接口统一，目前先这样处理 by 涂飞
	ret, err := serializeReturn(serializer, resp)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
