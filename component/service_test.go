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

package component

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/conn/message"
	"github.com/topfreegames/pitaya/constants"
)

type unexportedTestType struct {
	Base
}

type ExportedTypeWithNoHandlerAndNoRemote struct {
	Base
}

var tables = []struct {
	name     string
	comp     Component
	err      error
	handlers []string
	remotes  []string
}{
	{
		"valid",
		&TestType{},
		nil,
		[]string{"ExportedHandlerWithOnlySession", "ExportedHandlerWithSessionAndPointerWithPointerOut", "ExportedHandlerWithSessionAndPointerWithRawOut", "ExportedHandlerWithSessionAndRawWithNoOuts"},
		[]string{"ExportedRemotePointerOut", "ExportedRemoteRawOut"},
	},
	{"invalid", &unexportedTestType{}, errors.New("type unexportedTestType is not exported"), nil, nil},
	{"invalid", &ExportedTypeWithNoHandlerAndNoRemote{}, errors.New("type ExportedTypeWithNoHandlerAndNoRemote has no exported methods of handler type"), nil, nil},
}

func TestNewService(t *testing.T) {
	tables := []struct {
		name string
		opts []Option
	}{
		{"with-options", []Option{WithName("bla")}},
		{"without-options", []Option{}},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			s := NewService(&Base{}, table.opts)
			assert.NotNil(t, s)
		})
	}
}

func TestExtractHandler(t *testing.T) {
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			svc := NewService(table.comp, []Option{})
			err := svc.ExtractHandler()
			if table.err != nil {
				assert.EqualError(t, table.err, err.Error())
			} else {
				assert.NoError(t, err)
				for _, h := range table.handlers {
					val, ok := svc.Handlers[h]
					assert.True(t, ok)
					assert.NotNil(t, val)

				}
			}
		})
	}
}

func TestExtractRemote(t *testing.T) {
	tables[2].err = errors.New("type ExportedTypeWithNoHandlerAndNoRemote has no exported methods of remote type")
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			svc := NewService(table.comp, []Option{})
			err := svc.ExtractRemote()
			if table.err != nil {
				assert.EqualError(t, table.err, err.Error())
			} else {
				assert.NoError(t, err)
				for _, r := range table.remotes {
					val, ok := svc.Remotes[r]
					assert.True(t, ok)
					assert.NotNil(t, val)

				}
			}
		})
	}
}

func TestValidateMessageType(t *testing.T) {
	mtTables := []struct {
		name       string
		methodName string
		tp         message.Type
		msgType    message.Type
		err        error
		exit       bool
	}{
		{"notify-msg-to-notify", "ExportedHandlerWithOnlySession", message.Notify, message.Notify, nil, false},
		{"request-msg-to-request", "ExportedHandlerWithSessionAndPointerWithRawOut", message.Request, message.Request, nil, false},
		{"request-msg-to-notify", "ExportedHandlerWithOnlySession", message.Notify, message.Request, constants.ErrRequestOnNotify, true},
		{"request-msg-to-request", "ExportedHandlerWithSessionAndPointerWithRawOut", message.Request, message.Notify, constants.ErrNotifyOnRequest, false},
	}
	tObj := &TestType{}
	svc := NewService(tObj, []Option{})
	svc.ExtractHandler()
	svc.ExtractRemote()
	for _, table := range mtTables {
		t.Run(table.methodName, func(t *testing.T) {
			assert.Equal(t, table.tp, svc.Handlers[table.methodName].MessageType)
			exit, err := svc.Handlers[table.methodName].ValidateMessageType(table.msgType)
			if table.err != nil {
				assert.EqualError(t, table.err, err.Error())
			}
			assert.Equal(t, table.exit, exit)
		})
	}
}
