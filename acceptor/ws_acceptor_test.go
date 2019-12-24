package acceptor

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/conn/packet"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/helpers"
)

var wsAcceptorTables = []struct {
	name     string
	addr     string
	write    []byte
	certs    []string
	panicErr error
}{
	// TODO change to allocatable ports
	{"test_1", "0.0.0.0:0", []byte{0x01, 0x02}, []string{"./fixtures/server.crt", "./fixtures/server.key"}, nil},
	{"test_2", "127.0.0.1:0", []byte{0x00}, []string{"./fixtures/server.crt", "./fixtures/server.key"}, nil},
	{"test_3", "0.0.0.0:0", []byte{0x00}, []string{"wqodij"}, constants.ErrInvalidCertificates},
	{"test_4", "0.0.0.0:0", []byte{0x00}, []string{"wqodij", "qwdo", "wod"}, constants.ErrInvalidCertificates},
	{"test_4", "0.0.0.0:0", []byte{0x00}, []string{}, nil},
}

func TestNewWSAcceptor(t *testing.T) {
	t.Parallel()
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			if table.panicErr != nil {
				assert.PanicsWithValue(t, table.panicErr, func() {
					NewWSAcceptor(table.addr, table.certs...)
				})
			} else {
				var w *WSAcceptor
				assert.NotPanics(t, func() {
					w = NewWSAcceptor(table.addr, table.certs...)
				})

				if len(table.certs) == 2 {
					assert.Equal(t, table.certs[0], w.certFile)
					assert.Equal(t, table.certs[1], w.keyFile)
				} else {
					assert.Equal(t, "", w.certFile)
					assert.Equal(t, "", w.keyFile)
				}
				assert.NotNil(t, w)
			}
		})
	}
}

func TestWSAcceptorGetAddr(t *testing.T) {
	t.Parallel()
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := NewWSAcceptor(table.addr)
			// will return empty string because acceptor is not listening
			assert.Equal(t, "", w.GetAddr())
		})
	}
}

func TestWSAcceptorGetConn(t *testing.T) {
	t.Parallel()
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := NewWSAcceptor(table.addr)
			assert.NotNil(t, w.GetConnChan())
		})
	}
}

func mustConnectToWS(t *testing.T, write []byte, w *WSAcceptor, protocol string) {
	t.Helper()
	helpers.ShouldEventuallyReturn(t, func() error {
		addr := fmt.Sprintf("%s://%s", protocol, w.GetAddr())
		dialer := websocket.DefaultDialer
		conn, _, err := dialer.Dial(addr, nil)
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		conn.WriteMessage(websocket.BinaryMessage, write)
		defer conn.Close()
		return err
	}, nil, 10*time.Millisecond, 100*time.Millisecond)
}

func TestWSAcceptorListenAndServe(t *testing.T) {
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := NewWSAcceptor(table.addr)
			c := w.GetConnChan()
			defer w.Stop()
			go w.ListenAndServe()
			mustConnectToWS(t, table.write, w, "ws")
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*WSConn)
			defer conn.Close()
			assert.NotNil(t, conn)
		})
	}
}

func TestWSAcceptorListenAndServeTLS(t *testing.T) {
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := NewWSAcceptor(table.addr)
			c := w.GetConnChan()
			defer w.Stop()
			go w.ListenAndServeTLS("./fixtures/server.crt", "./fixtures/server.key")
			mustConnectToWS(t, table.write, w, "wss")
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*WSConn)
			defer conn.Close()
			assert.NotNil(t, conn)
		})
	}
}

func TestWSAcceptorStop(t *testing.T) {
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := NewWSAcceptor(table.addr)
			go w.ListenAndServe()
			mustConnectToWS(t, table.write, w, "ws")
			addr := fmt.Sprintf("ws://%s", w.GetAddr())
			w.Stop()
			_, _, err := websocket.DefaultDialer.Dial(addr, nil)
			assert.Error(t, err)
		})
	}
}

func TestWSConnRead(t *testing.T) {
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := NewWSAcceptor(table.addr)
			c := w.GetConnChan()
			defer w.Stop()
			go w.ListenAndServe()
			mustConnectToWS(t, table.write, w, "ws")
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*WSConn)
			defer conn.Close()
			b := make([]byte, len(table.write))
			n, err := conn.Read(b)
			assert.NoError(t, err)
			assert.Equal(t, len(table.write), n)
			assert.Equal(t, table.write, b)
		})
	}
}

func TestWSConnWrite(t *testing.T) {
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := NewWSAcceptor(table.addr)
			c := w.GetConnChan()
			defer w.Stop()
			go w.ListenAndServe()
			mustConnectToWS(t, table.write, w, "ws")
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*WSConn)
			defer conn.Close()
			b := make([]byte, len(table.write))
			n, err := conn.Write(b)
			assert.NoError(t, err)
			assert.Equal(t, len(table.write), n)
		})
	}
}

func TestWSConnLocalAddr(t *testing.T) {
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := NewWSAcceptor(table.addr)
			c := w.GetConnChan()
			defer w.Stop()
			go w.ListenAndServe()
			mustConnectToWS(t, table.write, w, "ws")
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*WSConn)
			defer conn.Close()
			a := conn.LocalAddr().String()
			assert.NotEmpty(t, a)
		})
	}
}

func TestWSConnRemoteAddr(t *testing.T) {
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := NewWSAcceptor(table.addr)
			c := w.GetConnChan()
			defer w.Stop()
			go w.ListenAndServe()
			mustConnectToWS(t, table.write, w, "ws")
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*WSConn)
			defer conn.Close()
			a := conn.RemoteAddr().String()
			assert.NotEmpty(t, a)
		})
	}
}

func TestWSConnSetDeadline(t *testing.T) {
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := NewWSAcceptor(table.addr)
			c := w.GetConnChan()
			defer w.Stop()
			go w.ListenAndServe()
			mustConnectToWS(t, table.write, w, "ws")
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*WSConn)
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(5 * time.Millisecond))
			time.Sleep(10 * time.Millisecond)
			_, err := conn.Write(table.write)
			assert.Error(t, err)
		})
	}
}

func TestWSGetNextMessage(t *testing.T) {
	tables := []struct {
		name string
		data []byte
		err  error
	}{
		{"invalid_header", []byte{0x00, 0x00, 0x00, 0x00}, packet.ErrWrongPomeloPacketType},
		{"valid_message", []byte{0x02, 0x00, 0x00, 0x01, 0x00}, nil},
		{"invalid_message", []byte{0x02, 0x00, 0x00, 0x02, 0x00}, constants.ErrReceivedMsgSmallerThanExpected},
		{"invalid_header", []byte{0x02, 0x00}, packet.ErrInvalidPomeloHeader},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			w := NewWSAcceptor("0.0.0.0:0")
			c := w.GetConnChan()
			defer w.Stop()
			go w.ListenAndServe()

			var conn *websocket.Conn
			var err error
			helpers.ShouldEventuallyReturn(t, func() error {
				addr := fmt.Sprintf("%s://%s", "ws", w.GetAddr())
				dialer := websocket.DefaultDialer
				conn, _, err = dialer.Dial(addr, nil)
				return err
			}, nil, 10*time.Millisecond, 100*time.Millisecond)

			playerConn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*WSConn)
			defer playerConn.Close()
			err = conn.WriteMessage(websocket.BinaryMessage, table.data)
			assert.NoError(t, err)
			msg, err := playerConn.GetNextMessage()
			if table.err != nil {
				assert.EqualError(t, err, table.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, table.data, msg)
			}
		})
	}
}

func TestWSGetNextMessageSequentially(t *testing.T) {
	w := NewWSAcceptor("0.0.0.0:0")
	c := w.GetConnChan()
	defer w.Stop()
	go w.ListenAndServe()

	var conn *websocket.Conn
	var err error
	helpers.ShouldEventuallyReturn(t, func() error {
		addr := fmt.Sprintf("%s://%s", "ws", w.GetAddr())
		dialer := websocket.DefaultDialer
		conn, _, err = dialer.Dial(addr, nil)
		return err
	}, nil, 10*time.Millisecond, 100*time.Millisecond)

	playerConn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*WSConn)
	defer playerConn.Close()
	msg1 := []byte{0x01, 0x00, 0x00, 0x02, 0x01, 0x01}
	msg2 := []byte{0x02, 0x00, 0x00, 0x02, 0x05, 0x04}
	err = conn.WriteMessage(websocket.BinaryMessage, msg1)
	assert.NoError(t, err)
	err = conn.WriteMessage(websocket.BinaryMessage, msg2)
	assert.NoError(t, err)
	msg, err := playerConn.GetNextMessage()
	assert.NoError(t, err)
	assert.Equal(t, msg1, msg)
	msg, err = playerConn.GetNextMessage()
	assert.NoError(t, err)
	assert.Equal(t, msg2, msg)
}
