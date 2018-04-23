package acceptor

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
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
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*wsConn)
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
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*wsConn)
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
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*wsConn)
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
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*wsConn)
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
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*wsConn)
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
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*wsConn)
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
			conn := helpers.ShouldEventuallyReceive(t, c, 100*time.Millisecond).(*wsConn)
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(5 * time.Millisecond))
			time.Sleep(10 * time.Millisecond)
			_, err := conn.Write(table.write)
			assert.Error(t, err)
		})
	}
}
