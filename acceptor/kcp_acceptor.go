// Copyright (c) nano Author and TFG Co. All Rights Reserved.
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

	"github.com/topfreegames/pitaya/logger"
	kcp "github.com/xtaci/kcp-go"
)

// KCPAcceptor struct
type KCPAcceptor struct {
	addr     string
	connChan chan net.Conn
	listener net.Listener
	running  bool
}

// NewKCPAcceptor creates a new instance of kcp acceptor
func NewKCPAcceptor(addr string) *KCPAcceptor {
	return &KCPAcceptor{
		addr:     addr,
		connChan: make(chan net.Conn),
		running:  false,
	}
}

// GetAddr returns the addr the acceptor will listen on
func (a *KCPAcceptor) GetAddr() string {
	if a.listener != nil {
		return a.listener.Addr().String()
	}
	return ""
}

// GetConnChan gets a connection channel
func (a *KCPAcceptor) GetConnChan() chan net.Conn {
	return a.connChan
}

// Stop stops the acceptor
func (a *KCPAcceptor) Stop() {
	a.running = false
	a.listener.Close()
}

// ListenAndServe using kcp acceptor
func (a *KCPAcceptor) ListenAndServe() {
	listener, err := kcp.Listen(a.addr)
	if err != nil {
		logger.Log.Fatal(err)
	}
	a.listener = listener
	a.running = true
	a.serve()
}

func (a *KCPAcceptor) serve() {
	defer a.Stop()
	for a.running {
		conn, err := a.listener.Accept()
		if err != nil {
			logger.Log.Error(err.Error())
			continue
		}

		a.connChan <- conn
	}
}
