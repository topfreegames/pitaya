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

package route

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var tables = []struct {
	server   string
	service  string
	method   string
	outStr   string
	shortStr string
}{
	{"someserver", "someservice", "somemethod", "someserver.someservice.somemethod", "someservice.somemethod"},
	{"", "someservice", "somemethod", "someservice.somemethod", "someservice.somemethod"},
}

func TestNewRoute(t *testing.T) {
	t.Parallel()
	for _, table := range tables {
		t.Run(table.outStr, func(t *testing.T) {
			r := NewRoute(table.server, table.service, table.method)
			assert.NotNil(t, r)
			assert.Equal(t, table.server, r.SvType)
			assert.Equal(t, table.service, r.Service)
			assert.Equal(t, table.method, r.Method)
		})
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	for _, table := range tables {
		t.Run(table.outStr, func(t *testing.T) {
			r := NewRoute(table.server, table.service, table.method)
			assert.Equal(t, table.outStr, r.String())
		})
	}
}

func TestShort(t *testing.T) {
	t.Parallel()
	for _, table := range tables {
		t.Run(table.outStr, func(t *testing.T) {
			r := NewRoute(table.server, table.service, table.method)
			assert.Equal(t, table.shortStr, r.Short())
		})
	}
}

func TestDecode(t *testing.T) {
	t.Parallel()
	dTables := []struct {
		route   string
		server  string
		service string
		method  string
		invalid error
	}{
		{"sv.some.method", "sv", "some", "method", nil},
		{"some.method", "", "some", "method", nil},
		{"invalid", "", "some", "method", ErrInvalidRoute},
		{"invalidstr..invalidmethod", "", "some", "method", ErrRouteFieldCantEmpty},
	}

	for _, table := range dTables {
		t.Run(table.route, func(t *testing.T) {
			r, err := Decode(table.route)
			if table.invalid == nil {
				assert.Equal(t, table.server, r.SvType)
				assert.Equal(t, table.service, r.Service)
				assert.Equal(t, table.method, r.Method)
			} else {
				assert.EqualError(t, err, table.invalid.Error())
			}
		})
	}
}
