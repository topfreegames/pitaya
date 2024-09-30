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

	"github.com/nats-io/nuid"

	"github.com/topfreegames/pitaya/v3/pkg/conn/message"
	"github.com/topfreegames/pitaya/v3/pkg/constants"
	pcontext "github.com/topfreegames/pitaya/v3/pkg/context"
	"github.com/topfreegames/pitaya/v3/pkg/logger"
	"github.com/topfreegames/pitaya/v3/pkg/logger/interfaces"
	"github.com/topfreegames/pitaya/v3/pkg/protos"
	"github.com/topfreegames/pitaya/v3/pkg/serialize"
	"github.com/topfreegames/pitaya/v3/pkg/tracing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func getLoggerFromArgs(args []reflect.Value) interfaces.Logger {
	for _, a := range args {
		if !a.IsValid() {
			continue
		}
		if ctx, ok := a.Interface().(context.Context); ok {
			logVal := ctx.Value(constants.LoggerCtxKey)
			if logVal != nil {
				log := logVal.(interfaces.Logger)
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
	return DefaultErrWrapper.Unmarshal(payload, serializer)
}

// GetErrorPayload creates and serializes an error payload
func GetErrorPayload(serializer serialize.Serializer, err error) ([]byte, error) {
	return DefaultErrWrapper.Marshal(err, serializer)
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
	requestID := pcontext.GetFromPropagateCtx(ctx, constants.RequestIDKey)
	if rID, ok := requestID.(string); ok {
		if rID == "" {
			requestID = nuid.New()
		}
	} else {
		requestID = nuid.New()
	}
	defaultLogger := logger.Log.WithFields(
		map[string]interface{}{
			"route":     route,
			"requestId": requestID,
			"userId":    userID,
		},
	)

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

	parent, err := tracing.ExtractSpan(ctx)
	if err != nil {
		return nil
	}

	ctx = trace.ContextWithRemoteSpanContext(ctx, parent)

	// Create a new context and span
	attributes := []attribute.KeyValue{
		attribute.String("local.id", serverID),
		attribute.String("span.kind", "server"),
	}
	peerId, ok := pcontext.GetFromPropagateCtx(ctx, constants.PeerIDKey).(string)
	if ok {
		attributes = append(attributes, attribute.String("peer.id", peerId))
	}
	peerService, ok := pcontext.GetFromPropagateCtx(ctx, constants.PeerServiceKey).(string)
	if ok {
		attributes = append(attributes, attribute.String("peer.service", peerService))
	}
	requestId, ok := pcontext.GetFromPropagateCtx(ctx, constants.RequestIDKey).(string)
	if ok {
		attributes = append(attributes, attribute.String("request.id", requestId))
	}

	ctx, _ = tracing.StartSpan(ctx, route, attributes...)

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

	requestID := pcontext.GetFromPropagateCtx(ctx, constants.RequestIDKey)
	if rID, ok := requestID.(string); !ok || (ok && rID == "") {
		requestID = nuid.New().Next()
		ctx = pcontext.AddToPropagateCtx(ctx, constants.RequestIDKey, requestID)
	}

	route := req.GetMsg().GetRoute()
	ctx = pcontext.AddToPropagateCtx(ctx, constants.RouteKey, route)
	ctx = CtxWithDefaultLogger(ctx, route, "")
	return ctx, nil
}
