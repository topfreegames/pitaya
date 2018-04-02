package acceptor_test

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/helpers"
)

var wsAcceptorTables = []struct {
	name   string
	addr   string
	wsPath string
}{
	// TODO change to allocatable ports
	{"test_1", "0.0.0.0:0", "/path"},
	{"test_2", "0.0.0.0:0", "/bla"},
	{"test_3", "0.0.0.0:0", "/test"},
	{"test_4", "127.0.0.1:0", "/test2"},
}

func TestNewWSAcceptor(t *testing.T) {
	t.Parallel()
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := acceptor.NewWSAcceptor(table.addr, table.wsPath)
			assert.NotNil(t, w)
		})
	}
}

func TestWSAcceptorGetAddr(t *testing.T) {
	t.Parallel()
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := acceptor.NewWSAcceptor(table.addr, table.wsPath)
			// will return empty string because acceptor is not listening
			assert.Equal(t, w.GetAddr(), "")
		})
	}
}

func TestWSAcceptorGetConn(t *testing.T) {
	t.Parallel()
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := acceptor.NewWSAcceptor(table.addr, table.wsPath)
			assert.NotNil(t, w.GetConnChan())
		})
	}
}

func TestWSAcceptorListenAndServe(t *testing.T) {
	for _, table := range wsAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			w := acceptor.NewWSAcceptor(table.addr, table.wsPath)
			//c := w.GetConnChan()
			//defer w.Stop()
			go w.ListenAndServe()
			helpers.ShouldEventuallyReturn(t, func() error {
				_, err := net.Dial("tcp", w.GetAddr())
				return err
			}, nil, 10*time.Millisecond, 100*time.Millisecond)
			//conn := <-c
			//assert.NotNil(t, conn)
		})
	}
}
