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

package acceptor

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/helpers"
)

var tcpAcceptorTables = []struct {
	name string
	addr string
}{
	{"test_1", "0.0.0.0:0"},
	{"test_4", "127.0.0.1:0"},
}

func TestNewTCPAcceptorGetConnChanAndGetAddr(t *testing.T) {
	t.Parallel()
	for _, table := range tcpAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			a := NewTCPAcceptor(table.addr)
			assert.NotNil(t, a)
		})
	}
}

func TestGetAddr(t *testing.T) {
	t.Parallel()
	for _, table := range tcpAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			a := NewTCPAcceptor(table.addr)
			// returns nothing because not listening yet
			assert.Equal(t, "", a.GetAddr())
		})
	}

}

func TestGetConnChan(t *testing.T) {
	t.Parallel()
	for _, table := range tcpAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			a := NewTCPAcceptor(table.addr)
			assert.NotNil(t, a.GetConnChan())
		})
	}
}

func TestListenAndServer(t *testing.T) {
	for _, table := range tcpAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			a := NewTCPAcceptor(table.addr)
			defer a.Stop()
			c := a.GetConnChan()
			go a.ListenAndServe()
			// should be able to connect within 100 milliseconds
			helpers.ShouldEventuallyReturn(t, func() error {
				n, err := net.Dial("tcp", a.GetAddr())
				defer n.Close()
				return err
			}, nil, 10*time.Millisecond, 100*time.Millisecond)
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond)
			assert.NotNil(t, conn)
		})
	}
}

func TestStop(t *testing.T) {
	for _, table := range tcpAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			a := NewTCPAcceptor(table.addr)
			go a.ListenAndServe()
			// should be able to connect within 100 milliseconds
			helpers.ShouldEventuallyReturn(t, func() error {
				_, err := net.Dial("tcp", a.GetAddr())
				return err
			}, nil, 10*time.Millisecond, 100*time.Millisecond)
			a.Stop()
			_, err := net.Dial("tcp", table.addr)
			assert.Error(t, err)
		})
	}
}
