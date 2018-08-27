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

package docgenerator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/protos/test"
)

type MyComp struct {
	component.Base
}

type MyStruct struct {
	Str             string
	privateInt      int
	Int             int       `json:"int"`
	Time            time.Time `json:"time"`
	NotOnJSONString string    `json:"-"`
	Bytes           []byte    `json:"bytes"`
	Struct          *struct {
		Int        int `json:"int"`
		NotPointer struct {
			Str string `json:"str"`
			Int int
		} `json:"notPointer"`
	} `json:"struct"`
	Slice []*struct {
		Int int `json:"int"`
		Str string
	} `json:"slice"`
	NotPointer struct {
		Str string `json:"str"`
		Int int
	} `json:"notPointer"`
}

func (m *MyComp) Init() {}

func (m *MyComp) Shutdown() {}

func (m *MyComp) HandlerEmpty(ctx context.Context) {}

func (m *MyComp) HandlerRaw(ctx context.Context, b []byte) ([]byte, error) {
	return nil, nil
}

func (m *MyComp) HandlerOrRemoteStruct(ctx context.Context, s *MyStruct) (*MyStruct, error) {
	s.privateInt = 0
	return nil, nil
}

func (m *MyComp) RemoteStruct(ctx context.Context, ss *test.SomeStruct) (*test.SomeStruct, error) {
	return nil, nil
}

func TestHandlersDoc(t *testing.T) {
	t.Parallel()

	handlerServices := map[string]*component.Service{}
	s := component.NewService(&MyComp{}, []component.Option{})
	err := s.ExtractHandler()
	assert.NoError(t, err)
	handlerServices[s.Name] = s

	doc, err := HandlersDocs("metagame", handlerServices, false)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"metagame.MyComp.HandlerEmpty": map[string]interface{}{
			"output": []interface{}{},
			"input":  interface{}(nil),
		},
		"metagame.MyComp.HandlerRaw": map[string]interface{}{
			"input":  "[]byte",
			"output": []interface{}{"[]byte", "error"},
		},
		"metagame.MyComp.HandlerOrRemoteStruct": map[string]interface{}{
			"input": map[string]interface{}{
				"bytes": "[]byte",
				"int":   "int",
				"notPointer": map[string]interface{}{
					"int": "int",
					"str": "string",
				},
				"str": "string",
				"struct": map[string]interface{}{
					"int": "int",
					"notPointer": map[string]interface{}{
						"int": "int",
						"str": "string",
					},
				},
				"slice": []interface{}{
					map[string]interface{}{
						"int": "int",
						"str": "string",
					},
				},
				"time": "time.Time",
			},
			"output": []interface{}{
				map[string]interface{}{
					"int": "int",
					"notPointer": map[string]interface{}{
						"Int": "int",
						"str": "string",
					},
					"Str": "string",
					"struct": map[string]interface{}{
						"int": "int",
						"notPointer": map[string]interface{}{
							"Int": "int",
							"str": "string",
						},
					},
					"slice": []interface{}{
						map[string]interface{}{
							"int": "int",
							"Str": "string",
						},
					},
					"time":  "time.Time",
					"bytes": "[]byte",
				},
				"error",
			},
		},
		"metagame.MyComp.RemoteStruct": map[string]interface{}{
			"input": map[string]interface{}{
				"A": "int32",
				"B": "string",
			},
			"output": []interface{}{
				map[string]interface{}{
					"A": "int32",
					"B": "string",
				},
				"error",
			},
		},
	}, doc)
}

func TestHandlersDocTrue(t *testing.T) {
	t.Parallel()

	handlerServices := map[string]*component.Service{}
	s := component.NewService(&MyComp{}, []component.Option{})
	err := s.ExtractHandler()
	assert.NoError(t, err)
	handlerServices[s.Name] = s

	doc, err := HandlersDocs("metagame", handlerServices, false)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"metagame.MyComp.HandlerOrRemoteStruct": map[string]interface{}{
			"input": map[string]interface{}{
				"time":  "time.Time",
				"bytes": "[]byte",
				"int":   "int",
				"notPointer": map[string]interface{}{
					"int": "int",
					"str": "string",
				},
				"slice": []interface{}{map[string]interface{}{
					"int": "int",
					"str": "string",
				},
				},
				"str": "string",
				"struct": map[string]interface{}{
					"int": "int",
					"notPointer": map[string]interface{}{
						"int": "int",
						"str": "string",
					},
				},
			},
			"output": []interface{}{map[string]interface{}{
				"Str":   "string",
				"bytes": "[]byte",
				"int":   "int",
				"notPointer": map[string]interface{}{
					"Int": "int",
					"str": "string",
				},
				"slice": []interface{}{map[string]interface{}{
					"Str": "string",
					"int": "int",
				},
				},
				"struct": map[string]interface{}{
					"int": "int",
					"notPointer": map[string]interface{}{
						"Int": "int",
						"str": "string",
					},
				},
				"time": "time.Time",
			},
				"error",
			},
		},
		"metagame.MyComp.HandlerRaw": map[string]interface{}{
			"input": "[]byte",
			"output": []interface{}{
				"[]byte",
				"error",
			},
		},
		"metagame.MyComp.RemoteStruct": map[string]interface{}{
			"input": map[string]interface{}{
				"A": "int32",
				"B": "string",
			},
			"output": []interface{}{map[string]interface{}{
				"A": "int32",
				"B": "string",
			},
				"error",
			},
		},
		"metagame.MyComp.HandlerEmpty": map[string]interface{}{
			"output": []interface{}{},
			"input":  interface{}(nil),
		},
	}, doc)
}

func TestRemotesDoc(t *testing.T) {
	t.Parallel()

	remoteServices := map[string]*component.Service{}
	s := component.NewService(&MyComp{}, []component.Option{})
	err := s.ExtractRemote()
	assert.NoError(t, err)
	remoteServices[s.Name] = s

	doc, err := RemotesDocs("metagame", remoteServices, false)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"metagame.MyComp.RemoteStruct": map[string]interface{}{
			"input": map[string]interface{}{
				"A": "int32",
				"B": "string",
			},
			"output": []interface{}{
				map[string]interface{}{
					"A": "int32",
					"B": "string",
				},
				"error",
			},
		},
	}, doc)
}

func TestRemotesDocTrue(t *testing.T) {
	t.Parallel()
	remoteServices := map[string]*component.Service{}
	s := component.NewService(&MyComp{}, []component.Option{})
	err := s.ExtractRemote()
	assert.NoError(t, err)
	remoteServices[s.Name] = s
	doc, err := RemotesDocs("metagame", remoteServices, true)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"metagame.MyComp.RemoteStruct": map[string]interface{}{
			"input": map[string]interface{}{
				"*test.SomeStruct": map[string]interface{}{

					"A": "int32",
					"B": "string",
				},
			},
			"output": []interface{}{
				map[string]interface{}{
					"*test.SomeStruct": map[string]interface{}{
						"A": "int32",
						"B": "string",
					}},
				"error",
			},
		},
	}, doc)
}
