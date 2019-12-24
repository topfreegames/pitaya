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

package acceptorwrapper

import (
	"github.com/topfreegames/pitaya/acceptor"
)

// BaseWrapper implements Wrapper by saving the acceptor as an attribute.
// Conns from acceptor.GetConnChan are processed by wrapConn and
// forwarded to its own connChan.
// Any new wrapper can inherit from BaseWrapper and just implement wrapConn.
type BaseWrapper struct {
	acceptor.Acceptor
	connChan chan acceptor.PlayerConn
	wrapConn func(acceptor.PlayerConn) acceptor.PlayerConn
}

// NewBaseWrapper returns an instance of BaseWrapper.
func NewBaseWrapper(wrapConn func(acceptor.PlayerConn) acceptor.PlayerConn) BaseWrapper {
	return BaseWrapper{
		connChan: make(chan acceptor.PlayerConn),
		wrapConn: wrapConn,
	}
}

// ListenAndServe starts a goroutine that wraps acceptor's conn
// and calls acceptor's listenAndServe
func (b *BaseWrapper) ListenAndServe() {
	go b.pipe()
	b.Acceptor.ListenAndServe()
}

// GetConnChan returns the wrapper conn chan
func (b *BaseWrapper) GetConnChan() chan acceptor.PlayerConn {
	return b.connChan
}

func (b *BaseWrapper) pipe() {
	for conn := range b.Acceptor.GetConnChan() {
		b.connChan <- b.wrapConn(conn)
	}
}
