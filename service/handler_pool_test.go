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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v2/component"
	"github.com/topfreegames/pitaya/v2/conn/message"
	"github.com/topfreegames/pitaya/v2/constants"
	e "github.com/topfreegames/pitaya/v2/errors"
	"github.com/topfreegames/pitaya/v2/pipeline"
	"github.com/topfreegames/pitaya/v2/protos/test"
	"github.com/topfreegames/pitaya/v2/route"
	"github.com/topfreegames/pitaya/v2/serialize/mocks"
	session_mocks "github.com/topfreegames/pitaya/v2/session/mocks"
)

func TestGetHandlerExists(t *testing.T) {
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	expected := &component.Handler{}
	handlerPool := NewHandlerPool()
	handlerPool.handlers[rt.Short()] = expected
	defer func() { delete(handlerPool.handlers, rt.Short()) }()

	h, err := handlerPool.getHandler(rt)
	assert.NoError(t, err)
	assert.Equal(t, expected, h)
}

func TestGetHandlerDoesntExist(t *testing.T) {
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	handlerPool := NewHandlerPool()
	h, err := handlerPool.getHandler(rt)
	assert.Nil(t, h)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("%s not found", rt.String()))
}

func TestProcessHandlerMessage(t *testing.T) {
	tObj := &TestType{}

	handlerPool := NewHandlerPool()

	m, ok := reflect.TypeOf(tObj).MethodByName("HandlerPointerRaw")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	handlerPool.handlers[rt.Short()] = &component.Handler{Receiver: reflect.ValueOf(tObj), Method: m, Type: m.Type.In(2)}

	m, ok = reflect.TypeOf(tObj).MethodByName("HandlerPointerErr")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rtErr := route.NewRoute("", uuid.New().String(), uuid.New().String())
	handlerPool.handlers[rtErr.Short()] = &component.Handler{Receiver: reflect.ValueOf(tObj), Method: m, Type: m.Type.In(2)}

	m, ok = reflect.TypeOf(tObj).MethodByName("HandlerPointerStruct")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rtSt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	handlerPool.handlers[rtSt.Short()] = &component.Handler{Receiver: reflect.ValueOf(tObj), Method: m, Type: m.Type.In(2)}

	tables := []struct {
		name         string
		route        *route.Route
		errSerReturn error
		errSerialize error
		outSerialize interface{}
		handlerType  message.Type
		msgType      interface{}
		remote       bool
		out          []byte
		err          error
	}{
		{"invalid_route", route.NewRoute("", "no", "no"), nil, nil, nil, message.Request, nil, false, nil, e.NewError(errors.New("pitaya/handler: no.no not found"), e.ErrNotFoundCode)},
		{"invalid_msg_type", rt, nil, nil, nil, message.Request, nil, false, nil, e.NewError(errInvalidMsg, e.ErrInternalCode)},
		{"request_on_notify", rt, nil, nil, nil, message.Notify, message.Request, false, nil, e.NewError(constants.ErrRequestOnNotify, e.ErrBadRequestCode)},
		{"failed_handle_args_unmarshal", rt, nil, errors.New("some error"), &test.SomeStruct{}, message.Request, message.Request, false, nil, e.NewError(errors.New("some error"), e.ErrBadRequestCode)},
		{"failed_pcall", rtErr, nil, nil, &test.SomeStruct{A: 1, B: "ok"}, message.Request, message.Request, false, nil, errors.New("HandlerPointerErr")},
		{"failed_serialize_return", rtSt, errors.New("ser ret error"), nil, &test.SomeStruct{A: 1, B: "ok"}, message.Request, message.Request, false, []byte("failed"), nil},
		{"ok", rt, nil, nil, &test.SomeStruct{}, message.Request, message.Request, false, []byte("ok"), nil},
		{"notify_on_request", rt, nil, nil, &test.SomeStruct{}, message.Request, message.Notify, false, []byte("ok"), nil},
		{"remote_notify", rt, nil, nil, &test.SomeStruct{}, message.Notify, message.Notify, true, []byte("ack"), nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			handlerPool.handlers[rt.Short()].MessageType = table.handlerType
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ss := session_mocks.NewMockSession(ctrl)
			ss.EXPECT().UID().Return("uid").AnyTimes()
			ss.EXPECT().ID().Return(int64(1)).AnyTimes()
			mockSerializer := mocks.NewMockSerializer(ctrl)
			if table.outSerialize != nil {
				mockSerializer.EXPECT().Unmarshal(gomock.Any(), gomock.Any()).Return(table.errSerialize).Do(
					func(p []byte, arg interface{}) {
						arg = table.outSerialize
					})

				if table.errSerReturn != nil {
					mockSerializer.EXPECT().Marshal(gomock.Any()).Return(table.out, table.errSerReturn)
					mockSerializer.EXPECT().Marshal(gomock.Any()).Return(table.out, nil)
				}
			}
			handlerHooks := pipeline.NewHandlerHooks()
			out, err := handlerPool.ProcessHandlerMessage(nil, table.route, mockSerializer, handlerHooks, ss, nil, table.msgType, table.remote)
			assert.Equal(t, table.out, out)
			assert.Equal(t, table.err, err)
		})
	}
}

func TestProcessHandlerMessageBrokenBeforePipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	handlerPool := NewHandlerPool()
	handlerPool.handlers[rt.Short()] = &component.Handler{}
	expected := errors.New("oh noes")
	before := func(ctx context.Context, in interface{}) (context.Context, interface{}, error) {
		return ctx, nil, expected
	}
	beforeHandler := pipeline.NewChannel()
	beforeHandler.PushFront(before)

	handlerHooks := pipeline.NewHandlerHooks()
	handlerHooks.BeforeHandler = beforeHandler
	ss := session_mocks.NewMockSession(ctrl)
	ss.EXPECT().UID().Return("uid").AnyTimes()
	ss.EXPECT().ID().Return(int64(1)).AnyTimes()
	out, err := handlerPool.ProcessHandlerMessage(nil, rt, nil, handlerHooks, ss, nil, message.Request, false)
	assert.Nil(t, out)
	assert.Equal(t, expected, err)
}

func TestProcessHandlerMessageBrokenAfterPipeline(t *testing.T) {
	tObj := &TestType{}
	m, ok := reflect.TypeOf(tObj).MethodByName("HandlerPointerRaw")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	handlerPool := NewHandlerPool()
	handlerPool.handlers[rt.Short()] = &component.Handler{Receiver: reflect.ValueOf(tObj), Method: m, Type: m.Type.In(2)}

	after := func(ctx context.Context, out interface{}, err error) (interface{}, error) {
		return nil, errors.New("oh noes")
	}
	afterHandler := pipeline.NewAfterChannel()
	afterHandler.PushFront(after)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ss := session_mocks.NewMockSession(ctrl)
	ss.EXPECT().UID().Return("uid").AnyTimes()
	ss.EXPECT().ID().Return(int64(1)).AnyTimes()

	mockSerializer := mocks.NewMockSerializer(ctrl)
	mockSerializer.EXPECT().Unmarshal(gomock.Any(), gomock.Any()).Return(nil).Do(
		func(p []byte, arg interface{}) {
			arg = &test.SomeStruct{}
		})

	handlerHooks := pipeline.NewHandlerHooks()
	handlerHooks.AfterHandler = afterHandler
	out, err := handlerPool.ProcessHandlerMessage(nil, rt, mockSerializer, handlerHooks, ss, nil, message.Request, false)
	assert.Nil(t, out)
	assert.Equal(t, errors.New("oh noes"), err)
}
