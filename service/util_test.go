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
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/helpers"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/pipeline"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/serialize/mocks"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/util"
)

var update = flag.Bool("update", false, "update .golden files")

type TestType struct {
	component.Base
}

type SomeStruct struct {
	A int
	B string
}

func (t *TestType) HandlerNil(*session.Session)                       {}
func (t *TestType) HandlerRaw(s *session.Session, msg []byte)         {}
func (t *TestType) HandlerPointer(s *session.Session, ss *SomeStruct) {}
func (t *TestType) HandlerPointerRaw(s *session.Session, ss *SomeStruct) ([]byte, error) {
	return []byte("ok"), nil
}
func (t *TestType) HandlerPointerStruct(s *session.Session, ss *SomeStruct) (*SomeStruct, error) {
	return &SomeStruct{A: 1, B: "ok"}, nil
}
func (t *TestType) HandlerPointerErr(s *session.Session, ss *SomeStruct) ([]byte, error) {
	return nil, errors.New("HandlerPointerErr")
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	gob.Register(SomeStruct{})
}

func shutdown() {}

func TestGetHandlerExists(t *testing.T) {
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	expected := &component.Handler{}
	handlers[rt.Short()] = expected
	defer func() { delete(handlers, rt.Short()) }()

	h, err := getHandler(rt)
	assert.NoError(t, err)
	assert.Equal(t, expected, h)
}

func TestGetHandlerDoesntExist(t *testing.T) {
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	h, err := getHandler(rt)
	assert.Nil(t, h)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("%s not found", rt.String()))
}

func TestUnmarshalHandlerArg(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name        string
		handlerName string
		isRawArg    bool
		payload     []byte
		out         interface{}
		err         error
	}{
		{"raw_arg", "HandlerRaw", true, []byte("hello"), []byte("hello"), nil},
		{"nil_handler", "HandlerNil", false, []byte("hello"), nil, nil},
		{"struct_handler", "HandlerPointer", false, []byte("hello"), &SomeStruct{}, nil},
		{"struct_handler_err", "HandlerPointer", false, []byte("hello"), nil, errors.New("some error")},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSerializer := mocks.NewMockSerializer(ctrl)

			tObj := &TestType{}
			m, ok := reflect.TypeOf(tObj).MethodByName(table.handlerName)
			assert.True(t, ok)
			assert.NotNil(t, m)
			handler := &component.Handler{
				Method:   m,
				IsRawArg: table.isRawArg,
			}
			mt := m.Type
			if mt.NumIn() == 3 {
				handler.Type = mt.In(2)
			}

			if !table.isRawArg && handler.Type != nil {
				mockSerializer.EXPECT().Unmarshal(
					table.payload,
					reflect.New(handler.Type.Elem()).Interface(),
				).Do(func(p []byte, arg interface{}) {
					arg = table.out
				}).Return(table.err)
			}

			arg, err := unmarshalHandlerArg(handler, mockSerializer, table.payload)
			assert.Equal(t, table.err, err)
			assert.Equal(t, table.out, arg)
		})
	}
}

func TestUnmarshalRemoteArg(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name string
		args []interface{}
	}{
		{"unmarshal_remote_test_1", []interface{}{[]byte{1}, "test", 1}},
		{"unmarshal_remote_test_2", []interface{}{[]byte{1}, SomeStruct{A: 1, B: "aaa"}, 34}},
		{"unmarshal_remote_test_3", []interface{}{"aaa"}},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			gp := filepath.Join("fixtures", table.name+".golden")
			if *update {
				b, err := util.GobEncode(table.args...)
				require.NoError(t, err)
				t.Log("updating golden file")
				helpers.WriteFile(t, gp, b)
			}
			payload := helpers.ReadFile(t, gp)

			args, err := unmarshalRemoteArg(payload)
			assert.NoError(t, err)
			assert.Equal(t, table.args, args)
		})
	}
}

func TestUnmarshalRemoteArgErr(t *testing.T) {
	t.Parallel()
	args, err := unmarshalRemoteArg([]byte(nil))
	assert.Empty(t, args)
	assert.Equal(t, errors.New("EOF"), err)
}

func TestGetMsgType(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name    string
		in      interface{}
		msgType message.Type
		err     error
	}{
		{"request", message.Request, message.Request, nil},
		{"notify", message.Notify, message.Notify, nil},
		{"response", message.Response, message.Response, nil},
		{"push", message.Push, message.Push, nil},
		{"protos_request", protos.MsgType_MsgRequest, message.Request, nil},
		{"protos_notify", protos.MsgType_MsgNotify, message.Notify, nil},
		{"invalid", "oops", message.Request, errInvalidMsg},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			msgType, err := getMsgType(table.in)
			assert.Equal(t, table.err, err)
			assert.Equal(t, table.msgType, msgType)
		})
	}
}

func TestExecuteBeforePipelineEmpty(t *testing.T) {
	expected := []byte("ok")
	res, err := executeBeforePipeline(nil, expected)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func TestExecuteBeforePipelineSuccess(t *testing.T) {
	ss := session.New(nil, false)
	data := []byte("ok")
	expected1 := []byte("oh noes 1")
	expected2 := []byte("oh noes 2")
	before1 := func(s *session.Session, in []byte) ([]byte, error) {
		assert.Equal(t, ss, s)
		assert.Equal(t, data, in)
		return expected1, nil
	}
	before2 := func(s *session.Session, in []byte) ([]byte, error) {
		assert.Equal(t, ss, s)
		assert.Equal(t, expected1, in)
		return expected2, nil
	}
	pipeline.BeforeHandler.PushBack(before1)
	pipeline.BeforeHandler.PushBack(before2)
	defer pipeline.BeforeHandler.Clear()

	res, err := executeBeforePipeline(ss, data)
	assert.NoError(t, err)
	assert.Equal(t, expected2, res)
}

func TestExecuteBeforePipelineError(t *testing.T) {
	ss := session.New(nil, false)
	expected := errors.New("oh noes")
	before := func(s *session.Session, in []byte) ([]byte, error) {
		assert.Equal(t, ss, s)
		return nil, expected
	}
	pipeline.BeforeHandler.PushFront(before)
	defer pipeline.BeforeHandler.Clear()

	_, err := executeBeforePipeline(ss, []byte("ok"))
	assert.Equal(t, expected, err)
}

func TestExecuteAfterPipelineEmpty(t *testing.T) {
	expected := []byte("whatever")
	res := executeAfterPipeline(nil, nil, expected)
	assert.Equal(t, expected, res)
}

func TestExecuteAfterPipelineSuccess(t *testing.T) {
	ss := session.New(nil, false)
	data := []byte("ok")
	expected1 := []byte("oh noes 1")
	expected2 := []byte("oh noes 2")
	after1 := func(s *session.Session, in []byte) ([]byte, error) {
		assert.Equal(t, ss, s)
		assert.Equal(t, data, in)
		return expected1, nil
	}
	after2 := func(s *session.Session, in []byte) ([]byte, error) {
		assert.Equal(t, ss, s)
		assert.Equal(t, expected1, in)
		return expected2, nil
	}
	pipeline.AfterHandler.PushBack(after1)
	pipeline.AfterHandler.PushBack(after2)
	defer pipeline.AfterHandler.Clear()

	res := executeAfterPipeline(ss, nil, []byte("ok"))
	assert.Equal(t, expected2, res)
}

func TestExecuteAfterPipelineError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSerializer := mocks.NewMockSerializer(ctrl)

	ss := session.New(nil, false)
	after := func(s *session.Session, in []byte) ([]byte, error) {
		assert.Equal(t, ss, s)
		return nil, errors.New("oh noes")
	}
	pipeline.AfterHandler.PushFront(after)
	defer pipeline.AfterHandler.Clear()

	expected := []byte("error")
	mockSerializer.EXPECT().Marshal(gomock.Any()).Return(expected, nil)
	res := executeAfterPipeline(ss, mockSerializer, []byte("ok"))
	assert.Equal(t, expected, res)
}

func TestSerializeReturn(t *testing.T) {
	tables := []struct {
		name               string
		isRawArg           bool
		in                 interface{}
		out                []byte
		errSerialize       error
		errGetErrorPayload error
	}{
		{"raw_arg", true, []byte("hello"), []byte("hello"), nil, nil},
		{"success", false, SomeStruct{A: 1, B: "hello"}, []byte("hello"), nil, nil},
		{"serialize_fail", false, SomeStruct{A: 1, B: "hello"}, nil, errors.New("some error"), nil},
		{"serialize_fail_err_payload", false, SomeStruct{A: 1, B: "hello"}, nil, errors.New("some error"), errors.New("some other error")},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSerializer := mocks.NewMockSerializer(ctrl)
			if !table.isRawArg {
				mockSerializer.EXPECT().Marshal(gomock.Any()).Return(table.out, table.errSerialize)
				if table.errSerialize != nil {
					mockSerializer.EXPECT().Marshal(gomock.Any()).Return(table.out, table.errGetErrorPayload)
				}
			}
			out, err := serializeReturn(mockSerializer, table.in)
			assert.Equal(t, table.out, out)
			assert.Equal(t, table.errGetErrorPayload, err)
		})
	}
}

func TestProcessHandlerMessage(t *testing.T) {
	tObj := &TestType{}

	m, ok := reflect.TypeOf(tObj).MethodByName("HandlerPointerRaw")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	handlers[rt.Short()] = &component.Handler{Receiver: reflect.ValueOf(tObj), Method: m, Type: m.Type.In(2)}

	m, ok = reflect.TypeOf(tObj).MethodByName("HandlerPointerErr")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rtErr := route.NewRoute("", uuid.New().String(), uuid.New().String())
	handlers[rtErr.Short()] = &component.Handler{Receiver: reflect.ValueOf(tObj), Method: m, Type: m.Type.In(2)}

	m, ok = reflect.TypeOf(tObj).MethodByName("HandlerPointerStruct")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rtSt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	handlers[rtSt.Short()] = &component.Handler{Receiver: reflect.ValueOf(tObj), Method: m, Type: m.Type.In(2)}
	defer func() { handlers = make(map[string]*component.Handler, 0) }()

	ss := session.New(nil, false)
	cs := reflect.ValueOf(ss)

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
		{"invalid_route", route.NewRoute("", "no", "no"), nil, nil, nil, message.Request, nil, false, nil, errors.New("pitaya/handler: no.no not found")},
		{"invalid_msg_type", rt, nil, nil, nil, message.Request, nil, false, nil, errors.New("invalid message type provided")},
		{"request_on_notify", rt, nil, nil, nil, message.Notify, message.Request, false, nil, errors.New("tried to request a notify route")},
		{"failed_handle_args_unmarshal", rt, nil, errors.New("some error"), &SomeStruct{}, message.Request, message.Request, false, nil, errors.New("some error")},
		{"failed_pcall", rtErr, nil, nil, &SomeStruct{A: 1, B: "ok"}, message.Request, message.Request, false, nil, errors.New("HandlerPointerErr")},
		{"failed_serialize_return", rtSt, errors.New("ser ret error"), nil, &SomeStruct{A: 1, B: "ok"}, message.Request, message.Request, false, []byte("failed"), nil},
		{"ok", rt, nil, nil, &SomeStruct{}, message.Request, message.Request, false, []byte("ok"), nil},
		{"notify_on_request", rt, nil, nil, &SomeStruct{}, message.Request, message.Notify, false, []byte("ok"), nil},
		{"remote_notify", rt, nil, nil, &SomeStruct{}, message.Notify, message.Notify, true, []byte("ack"), nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			handlers[rt.Short()].MessageType = table.handlerType
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
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
			out, err := processHandlerMessage(table.route, mockSerializer, cs, ss, nil, table.msgType, table.remote)
			assert.Equal(t, table.out, out)
			assert.Equal(t, table.err, err)
		})
	}
}

func TestProcessHandlerMessageBrokenBeforePipeline(t *testing.T) {
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	handlers[rt.Short()] = &component.Handler{}
	defer func() { delete(handlers, rt.Short()) }()
	expected := errors.New("oh noes")
	before := func(s *session.Session, in []byte) ([]byte, error) {
		return nil, expected
	}
	pipeline.BeforeHandler.PushFront(before)
	defer pipeline.BeforeHandler.Clear()

	ss := session.New(nil, false)
	cs := reflect.ValueOf(ss)
	out, err := processHandlerMessage(rt, nil, cs, ss, nil, message.Request, false)
	assert.Nil(t, out)
	assert.Equal(t, expected, err)
}

func TestProcessHandlerMessageBrokenAfterPipeline(t *testing.T) {
	tObj := &TestType{}
	m, ok := reflect.TypeOf(tObj).MethodByName("HandlerPointerRaw")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	handlers[rt.Short()] = &component.Handler{Receiver: reflect.ValueOf(tObj), Method: m, Type: m.Type.In(2)}
	defer func() { delete(handlers, rt.Short()) }()

	after := func(s *session.Session, in []byte) ([]byte, error) {
		return nil, errors.New("oh noes")
	}
	pipeline.AfterHandler.PushFront(after)
	defer pipeline.AfterHandler.Clear()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ss := session.New(nil, false)
	cs := reflect.ValueOf(ss)
	mockSerializer := mocks.NewMockSerializer(ctrl)
	mockSerializer.EXPECT().Unmarshal(gomock.Any(), gomock.Any()).Return(nil).Do(
		func(p []byte, arg interface{}) {
			arg = &SomeStruct{}
		})
	expected := []byte("oops")
	mockSerializer.EXPECT().Marshal(gomock.Any()).Return(expected, nil)

	out, err := processHandlerMessage(rt, mockSerializer, cs, ss, nil, message.Request, false)
	assert.Equal(t, expected, out)
	assert.NoError(t, err)
}
