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
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v2/conn/packet"
	"github.com/topfreegames/pitaya/v2/constants"
	"github.com/topfreegames/pitaya/v2/helpers"
)

var tcpAcceptorTables = []struct {
	name     string
	addr     string
	certs    []string
	panicErr string
}{
	{"test_1", "0.0.0.0:0", []string{"./fixtures/server.crt", "./fixtures/server.key"}, ""},
	{"test_2", "0.0.0.0:0", []string{}, ""},
	{"test_3", "127.0.0.1:0", []string{"wqd"}, "certificates must be exactly two"},
	{"test_4", "127.0.0.1:0", []string{"wqd", "wqdqwd"}, "invalid certificates: open wqd: no such file or directory"},
	{"test_5", "127.0.0.1:0", []string{"wqd", "wqdqwd", "wqdqdqwd"}, "certificates must be exactly two"},
}

func TestNewTCPAcceptorGetConnChanAndGetAddr(t *testing.T) {
	t.Parallel()
	for _, table := range tcpAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			if table.panicErr != "" {
				if table.name == "test_4" && runtime.GOOS == "windows" {
					table.panicErr = "invalid certificates: open wqd: The system cannot find the file specified."
				}
				assert.PanicsWithError(t, table.panicErr, func() {
					NewTCPAcceptor(table.addr, table.certs...)
				})
			} else {
				var a *TCPAcceptor
				assert.NotPanics(t, func() {
					a = NewTCPAcceptor(table.addr, table.certs...)
				})

				if len(table.certs) == 2 {
					assert.Len(t, a.certs, 1)
				} else {
					assert.Len(t, a.certs, 0)
				}
				assert.NotNil(t, a)
			}
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

func TestListenAndServe(t *testing.T) {
	for _, table := range tcpAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			a := NewTCPAcceptor(table.addr)
			defer a.Stop()
			c := a.GetConnChan()
			go a.ListenAndServe()
			// should be able to connect within 100 milliseconds
			helpers.ShouldEventuallyReturn(t, func() error {
				_, err := net.Dial("tcp", a.GetAddr())
				return err
			}, nil, 10*time.Millisecond, 100*time.Millisecond)
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond)
			assert.NotNil(t, conn)
		})
	}
}

func TestListenAndServeTLS(t *testing.T) {
	for _, table := range tcpAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			a := NewTCPAcceptor(table.addr)
			defer a.Stop()
			c := a.GetConnChan()

			go a.ListenAndServeTLS("./fixtures/server.crt", "./fixtures/server.key")
			// should be able to connect within 100 milliseconds
			helpers.ShouldEventuallyReturn(t, func() error {
				_, err := net.Dial("tcp", a.GetAddr())
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

func TestGetNextMessage(t *testing.T) {
	tables := []struct {
		name string
		data []byte
		err  error
	}{
		{"invalid_header", []byte{0x00, 0x00, 0x00, 0x00}, packet.ErrWrongPomeloPacketType},
		{"valid_message", []byte{0x02, 0x00, 0x00, 0x01, 0x00}, nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			a := NewTCPAcceptor("0.0.0.0:0")
			go a.ListenAndServe()
			defer a.Stop()
			c := a.GetConnChan()
			// should be able to connect within 100 milliseconds
			var conn net.Conn
			var err error
			helpers.ShouldEventuallyReturn(t, func() error {
				conn, err = net.Dial("tcp", a.GetAddr())
				return err
			}, nil, 10*time.Millisecond, 100*time.Millisecond)

			defer conn.Close()
			playerConn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(PlayerConn)
			_, err = conn.Write(table.data)
			assert.NoError(t, err)

			msg, err := playerConn.GetNextMessage()
			if table.err != nil {
				assert.EqualError(t, err, table.err.Error())
			} else {
				assert.Equal(t, table.data, msg)
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetNextMessageTwoMessagesInBuffer(t *testing.T) {
	a := NewTCPAcceptor("0.0.0.0:0")
	go a.ListenAndServe()
	defer a.Stop()
	c := a.GetConnChan()
	// should be able to connect within 100 milliseconds
	var conn net.Conn
	var err error
	helpers.ShouldEventuallyReturn(t, func() error {
		conn, err = net.Dial("tcp", a.GetAddr())
		return err
	}, nil, 10*time.Millisecond, 100*time.Millisecond)
	defer conn.Close()

	playerConn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(PlayerConn)
	msg1 := []byte{0x01, 0x00, 0x00, 0x01, 0x02}
	msg2 := []byte{0x02, 0x00, 0x00, 0x02, 0x01, 0x01}
	buffer := append(msg1, msg2...)
	_, err = conn.Write(buffer)
	assert.NoError(t, err)

	msg, err := playerConn.GetNextMessage()
	assert.NoError(t, err)
	assert.Equal(t, msg1, msg)

	msg, err = playerConn.GetNextMessage()
	assert.NoError(t, err)
	assert.Equal(t, msg2, msg)
}

func TestGetNextMessageEOF(t *testing.T) {
	a := NewTCPAcceptor("0.0.0.0:0")
	go a.ListenAndServe()
	defer a.Stop()
	c := a.GetConnChan()
	// should be able to connect within 100 milliseconds
	var conn net.Conn
	var err error
	helpers.ShouldEventuallyReturn(t, func() error {
		conn, err = net.Dial("tcp", a.GetAddr())
		return err
	}, nil, 10*time.Millisecond, 100*time.Millisecond)

	playerConn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(PlayerConn)
	buffer := []byte{0x02, 0x00, 0x00, 0x02, 0x01}
	_, err = conn.Write(buffer)
	assert.NoError(t, err)

	go func() {
		time.Sleep(100 * time.Millisecond)
		conn.Close()
	}()

	_, err = playerConn.GetNextMessage()
	assert.EqualError(t, err, constants.ErrReceivedMsgSmallerThanExpected.Error())
}

func TestGetNextMessageEmptyEOF(t *testing.T) {
	a := NewTCPAcceptor("0.0.0.0:0")
	go a.ListenAndServe()
	defer a.Stop()
	c := a.GetConnChan()
	// should be able to connect within 100 milliseconds
	var conn net.Conn
	var err error
	helpers.ShouldEventuallyReturn(t, func() error {
		conn, err = net.Dial("tcp", a.GetAddr())
		return err
	}, nil, 10*time.Millisecond, 100*time.Millisecond)

	playerConn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(PlayerConn)

	go func() {
		time.Sleep(100 * time.Millisecond)
		conn.Close()
	}()

	_, err = playerConn.GetNextMessage()
	assert.EqualError(t, err, constants.ErrConnectionClosed.Error())
}

func TestGetNextMessageInParts(t *testing.T) {
	a := NewTCPAcceptor("0.0.0.0:0")
	go a.ListenAndServe()
	defer a.Stop()
	c := a.GetConnChan()
	// should be able to connect within 100 milliseconds
	var conn net.Conn
	var err error
	helpers.ShouldEventuallyReturn(t, func() error {
		conn, err = net.Dial("tcp", a.GetAddr())
		return err
	}, nil, 10*time.Millisecond, 100*time.Millisecond)

	defer conn.Close()
	playerConn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(PlayerConn)
	part1 := []byte{0x02, 0x00, 0x00, 0x03, 0x01}
	part2 := []byte{0x01, 0x02}
	_, err = conn.Write(part1)
	assert.NoError(t, err)

	go func() {
		time.Sleep(200 * time.Millisecond)
		_, err = conn.Write(part2)
	}()

	msg, err := playerConn.GetNextMessage()
	assert.NoError(t, err)
	assert.Equal(t, msg, append(part1, part2...))

}
