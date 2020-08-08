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

package util

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime/debug"
	"strconv"

	"github.com/topfreegames/pitaya/conn/message"
	"github.com/topfreegames/pitaya/constants"
	pcontext "github.com/topfreegames/pitaya/context"
	e "github.com/topfreegames/pitaya/errors"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/serialize"
	"github.com/topfreegames/pitaya/serialize/json"
	"github.com/topfreegames/pitaya/serialize/protobuf"
	"github.com/topfreegames/pitaya/tracing"

	"github.com/google/uuid"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func getLoggerFromArgs(args []reflect.Value) logger.Logger {
	for _, a := range args {
		if !a.IsValid() {
			continue
		}
		if ctx, ok := a.Interface().(context.Context); ok {
			logVal := ctx.Value(constants.LoggerCtxKey)
			if logVal != nil {
				log := logVal.(logger.Logger)
				return log
			}
		}
	}
	return logger.Log
}

// Pcall calls a method that returns an interface and an error and recovers in case of panic
func Pcall(method reflect.Method, args []reflect.Value) (rets interface{}, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			// Try to use logger from context here to help trace error cause
			stackTrace := debug.Stack()
			stackTraceAsRawStringLiteral := strconv.Quote(string(stackTrace))
			log := getLoggerFromArgs(args)
			log.Errorf("panic - pitaya/dispatch: methodName=%s panicData=%v stackTrace=%s", method.Name, rec, stackTraceAsRawStringLiteral)

			if s, ok := rec.(string); ok {
				err = errors.New(s)
			} else {
				err = fmt.Errorf("rpc call internal error - %s: %v", method.Name, rec)
			}
		}
	}()

	r := method.Func.Call(args)
	// r can have 0 length in case of notify handlers
	// otherwise it will have 2 outputs: an interface and an error
	if len(r) == 2 {
		if v := r[1].Interface(); v != nil {
			err = v.(error)
		} else if !r[0].IsNil() {
			rets = r[0].Interface()
		} else {
			err = constants.ErrReplyShouldBeNotNull
		}
	}
	return
}

// SliceContainsString returns true if a slice contains the string
func SliceContainsString(slice []string, str string) bool {
	for _, value := range slice {
		if value == str {
			return true
		}
	}
	return false
}

// SerializeOrRaw serializes the interface if its not an array of bytes already
func SerializeOrRaw(serializer serialize.Serializer, v interface{}) ([]byte, error) {
	if data, ok := v.([]byte); ok {
		return data, nil
	}
	data, err := serializer.Marshal(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// FileExists tells if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

// GetErrorFromPayload gets the error from payload
func GetErrorFromPayload(serializer serialize.Serializer, payload []byte) error {
	err := &e.Error{Code: e.ErrUnknownCode}
	switch serializer.(type) {
	case *json.Serializer:
		_ = serializer.Unmarshal(payload, err)
	case *protobuf.Serializer:
		pErr := &protos.Error{Code: e.ErrUnknownCode}
		_ = serializer.Unmarshal(payload, pErr)
		err = &e.Error{Code: pErr.Code, Message: pErr.Msg, Metadata: pErr.Metadata}
	}
	return err
}

// GetErrorPayload creates and serializes an error payload
func GetErrorPayload(serializer serialize.Serializer, err error) ([]byte, error) {
	code := e.ErrUnknownCode
	msg := err.Error()
	metadata := map[string]string{}
	if val, ok := err.(*e.Error); ok {
		code = val.Code
		metadata = val.Metadata
	}
	errPayload := &protos.Error{
		Code: code,
		Msg:  msg,
	}
	if len(metadata) > 0 {
		errPayload.Metadata = metadata
	}
	return SerializeOrRaw(serializer, errPayload)
}

// ConvertProtoToMessageType converts a protos.MsgType to a message.Type
func ConvertProtoToMessageType(protoMsgType protos.MsgType) message.Type {
	var msgType message.Type
	switch protoMsgType {
	case protos.MsgType_MsgRequest:
		msgType = message.Request
	case protos.MsgType_MsgNotify:
		msgType = message.Notify
	}
	return msgType
}

// CtxWithDefaultLogger inserts a default logger on ctx to be used on handlers and remotes.
// If using logrus, userId, route and requestId will be added as fields.
// Otherwise the pitaya logger will be used as it is.
func CtxWithDefaultLogger(ctx context.Context, route, userID string) context.Context {
	var defaultLogger logger.Logger
	logrusLogger, ok := logger.Log.(logrus.FieldLogger)
	if ok {
		requestID := pcontext.GetFromPropagateCtx(ctx, constants.RequestIDKey)
		if rID, ok := requestID.(string); ok {
			if rID == "" {
				requestID = uuid.New()
			}
		} else {
			requestID = uuid.New()
		}
		defaultLogger = logrusLogger.WithFields(
			logrus.Fields{
				"route":     route,
				"requestId": requestID,
				"userId":    userID,
			})
	} else {
		defaultLogger = logger.Log
	}

	return context.WithValue(ctx, constants.LoggerCtxKey, defaultLogger)
}

// StartSpanFromRequest starts a tracing span from the request
func StartSpanFromRequest(
	ctx context.Context,
	serverID, route string,
) context.Context {
	if ctx == nil {
		return nil
	}
	tags := opentracing.Tags{
		"local.id":     serverID,
		"span.kind":    "server",
		"peer.id":      pcontext.GetFromPropagateCtx(ctx, constants.PeerIDKey),
		"peer.service": pcontext.GetFromPropagateCtx(ctx, constants.PeerServiceKey),
		"request.id":   pcontext.GetFromPropagateCtx(ctx, constants.RequestIDKey),
	}
	parent, err := tracing.ExtractSpan(ctx)
	if err != nil {
		logger.Log.Warnf("failed to retrieve parent span: %s", err.Error())
	}
	ctx = tracing.StartSpan(ctx, route, tags, parent)
	return ctx
}

// GetContextFromRequest gets the context from a request
func GetContextFromRequest(req *protos.Request, serverID string) (context.Context, error) {
	ctx, err := pcontext.Decode(req.GetMetadata())
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		return nil, constants.ErrNoContextFound
	}
	ctx = CtxWithDefaultLogger(ctx, req.GetMsg().GetRoute(), "")
	return ctx, nil
}
