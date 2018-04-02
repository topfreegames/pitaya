package acceptor_test

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/helpers"
)

var tcpAcceptorTables = []struct {
	name string
	addr string
}{
	{"test_1", "0.0.0.0:0"},
	{"test_2", "0.0.0.0:0"},
	{"test_3", "0.0.0.0:0"},
	{"test_4", "127.0.0.1:0"},
}

func TestNewTCPAcceptorGetConnChanAndGetAddr(t *testing.T) {
	t.Parallel()
	for _, table := range tcpAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			a := acceptor.NewTCPAcceptor(table.addr)
			assert.NotNil(t, a)
		})
	}
}

func TestGetAddr(t *testing.T) {
	t.Parallel()
	for _, table := range tcpAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			a := acceptor.NewTCPAcceptor(table.addr)
			// returns nothing because not listening yet
			assert.Equal(t, a.GetAddr(), "")
		})
	}

}

func TestGetConnChan(t *testing.T) {
	t.Parallel()
	for _, table := range tcpAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			a := acceptor.NewTCPAcceptor(table.addr)
			assert.NotNil(t, a.GetConnChan())
		})
	}
}

func TestListenAndServer(t *testing.T) {
	for _, table := range tcpAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			a := acceptor.NewTCPAcceptor(table.addr)
			defer a.Stop()
			c := a.GetConnChan()
			go a.ListenAndServe()
			// should be able to connect within 100 milliseconds
			helpers.ShouldEventuallyReturn(t, func() error {
				n, err := net.Dial("tcp", a.GetAddr())
				defer n.Close()
				return err
			}, nil, 10*time.Millisecond, 100*time.Millisecond)
			conn := <-c
			assert.NotNil(t, conn)
		})
	}
}

func TestStop(t *testing.T) {
	for _, table := range tcpAcceptorTables {
		t.Run(table.name, func(t *testing.T) {
			a := acceptor.NewTCPAcceptor(table.addr)
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
