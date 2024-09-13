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

package cluster

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var svTestTables = []struct {
	id       string
	svType   string
	metadata map[string]string
	frontend bool
}{
	{"someid-1", "somesvtype", map[string]string{"bla": "ola"}, true},
	{"someid-2", "somesvtype", nil, true},
	{"someid-3", "somesvtype", map[string]string{"sv": "game"}, false},
	{"someid-4", "somesvtype", map[string]string{}, false},
}

func TestNewServer(t *testing.T) {
	t.Parallel()
	for _, table := range svTestTables {
		t.Run(table.id, func(t *testing.T) {
			s := NewServer(table.id, table.svType, table.frontend, table.metadata)
			assert.Equal(t, table.id, s.ID)
			assert.Equal(t, table.metadata, s.Metadata)
			assert.Equal(t, table.frontend, s.Frontend)
			assert.NotNil(t, s.Hostname)
		})
	}
}

func TestAsJSONString(t *testing.T) {
	t.Parallel()
	for _, table := range svTestTables {
		t.Run(table.id, func(t *testing.T) {
			s := NewServer(table.id, table.svType, table.frontend, table.metadata)
			b, err := json.Marshal(s)
			assert.NoError(t, err)
			assert.Equal(t, string(b), s.AsJSONString())
		})
	}
}
