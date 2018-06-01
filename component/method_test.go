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
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/protos/test"
)

type TestType struct {
	Base
}

func (t *TestType) ExportedNoHandlerNorRemote()                                                {}
func (t *TestType) ExportedHandlerWithOnlySession(ctx context.Context)                         {}
func (t *TestType) ExportedHandlerWithSessionAndRawWithNoOuts(ctx context.Context, msg []byte) {}
func (t *TestType) ExportedHandlerWithSessionAndPointerWithRawOut(ctx context.Context, tt *TestType) ([]byte, error) {
	return nil, nil
}
func (t *TestType) ExportedHandlerWithSessionAndPointerWithPointerOut(ctx context.Context, tt *TestType) (*TestType, error) {
	return nil, nil
}
func (t *TestType) ExportedRemoteRawOut(ctx context.Context) (*test.SomeStruct, error) {
	return nil, nil
}
func (t *TestType) ExportedRemotePointerOut(ctx context.Context) (*test.SomeStruct, error) {
	return nil, nil
}

func TestIsExported(t *testing.T) {
	t.Parallel()
	tables := []struct {
		method string
		res    bool
	}{
		{"notExported", false},
		{"Exported", true},
	}

	for _, table := range tables {
		t.Run(table.method, func(t *testing.T) {
			assert.Equal(t, table.res, isExported(table.method))
		})
	}
}

func TestIsRemoteMethod(t *testing.T) {
	t.Parallel()
	tables := []struct {
		methodName string
		isRemote   bool
	}{
		{"ExportedNoHandlerNorRemote", false},
		{"ExportedHandlerWithOnlySession", false},
		{"ExportedHandlerWithSessionAndRawWithNoOuts", false},
		{"ExportedRemoteRawOut", true},
		{"ExportedRemotePointerOut", true},
	}

	for _, table := range tables {
		t.Run(table.methodName, func(t *testing.T) {
			tObj := &TestType{}
			m, ok := reflect.TypeOf(tObj).MethodByName(table.methodName)
			assert.True(t, ok)
			assert.NotNil(t, m)
			assert.Equal(t, table.isRemote, isRemoteMethod(m))
		})
	}
}

func TestIsHandleMethod(t *testing.T) {
	t.Parallel()
	tables := []struct {
		methodName string
		isRemote   bool
	}{
		{"ExportedNoHandlerNorRemote", false},
		{"ExportedHandlerWithOnlySession", true},
		{"ExportedHandlerWithSessionAndRawWithNoOuts", true},
		{"ExportedHandlerWithSessionAndPointerWithRawOut", true},
		{"ExportedHandlerWithSessionAndPointerWithPointerOut", true},
	}
	for _, table := range tables {
		t.Run(table.methodName, func(t *testing.T) {
			tObj := &TestType{}
			m, ok := reflect.TypeOf(tObj).MethodByName(table.methodName)
			assert.True(t, ok)
			assert.NotNil(t, m)
			assert.Equal(t, table.isRemote, isHandlerMethod(m))
		})
	}
}

func TestSuitableRemoteMethods(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name     string
		nameFunc func(string) string
		outKeys  []string
	}{
		{"noNameFunc", nil, []string{"ExportedRemotePointerOut", "ExportedRemoteRawOut"}},
		{"withNameFunc", strings.ToLower, []string{"exportedremotepointerout", "exportedremoterawout"}},
	}
	tObj := &TestType{}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			out := suitableRemoteMethods(reflect.TypeOf(tObj), table.nameFunc)
			for _, r := range table.outKeys {
				val, ok := out[r]
				assert.True(t, ok)
				assert.NotNil(t, val)
			}
		})
	}
}

func TestSuitableHandlerMethods(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name     string
		nameFunc func(string) string
		outKeys  []string
	}{
		{"noNameFunc", nil, []string{"ExportedHandlerWithOnlySession", "ExportedHandlerWithSessionAndRawWithNoOuts", "ExportedHandlerWithSessionAndPointerWithRawOut", "ExportedHandlerWithSessionAndPointerWithPointerOut"}},
		{"withNameFunc", strings.ToLower, []string{"exportedhandlerwithonlysession", "exportedhandlerwithsessionandrawwithnoouts", "exportedhandlerwithsessionandpointerwithrawout", "exportedhandlerwithsessionandpointerwithpointerout"}},
	}
	tObj := &TestType{}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			out := suitableHandlerMethods(reflect.TypeOf(tObj), table.nameFunc)
			for _, r := range table.outKeys {
				val, ok := out[r]
				assert.True(t, ok)
				assert.NotNil(t, val)

			}
		})
	}

}
