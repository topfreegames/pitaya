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
	"github.com/topfreegames/pitaya/v3/pkg/acceptor"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v3/pkg/mocks"
)

func TestListenAndServe(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAcceptor := mocks.NewMockAcceptor(ctrl)
	mockConn := mocks.NewMockPlayerConn(ctrl)

	conns := make(chan acceptor.PlayerConn)
	exit := make(chan struct{})
	reads := 3
	go func() {
		for i := 0; i < reads; i++ {
			mockConn.EXPECT().Read([]byte{})
			conns <- mockConn
		}
	}()

	mockAcceptor.EXPECT().GetConnChan().Return(conns)
	wrapper := &BaseWrapper{
		Acceptor: mockAcceptor,
		connChan: make(chan acceptor.PlayerConn),
		wrapConn: func(c acceptor.PlayerConn) acceptor.PlayerConn {
			_, err := c.Read([]byte{})
			assert.NoError(t, err)
			return c
		},
	}

	go func() {
		i := 0
		for range wrapper.GetConnChan() {
			i++
			if i == reads {
				close(exit)
			}
		}
	}()

	mockAcceptor.EXPECT().ListenAndServe().Do(func() { <-exit })
	wrapper.ListenAndServe()
}
