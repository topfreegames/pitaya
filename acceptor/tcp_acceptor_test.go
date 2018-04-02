package acceptor_test

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/helpers"
)

func TestNewTCPAcceptorGetConnChanAndGetAddr(t *testing.T) {
	tables := []struct {
		name string
		addr string
	}{
		{"table_test_1", "0.0.0.0:2515"},
		{"table_test_2", "0.0.0.0:2516"},
		{"table_test_3", "0.0.0.0:2517"},
		{"table_test_4", "127.0.0.1:2517"},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			a := acceptor.NewTCPAcceptor(table.addr)
			assert.NotNil(t, a.GetConnChan())
			assert.Equal(t, a.GetAddr(), table.addr)
		})
	}
}

func TestListenAndServer(t *testing.T) {
	tables := []struct {
		name string
		addr string
	}{
		{"test_listen_1", "0.0.0.0:13333"},
		{"test_listen_2", "0.0.0.0:13366"},
		{"test_listen_3", "0.0.0.0:12266"},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			a := acceptor.NewTCPAcceptor(table.addr)
			c := a.GetConnChan()
			go a.ListenAndServe()
			// should be able to connect within 100 milliseconds
			helpers.ShouldEventuallyReturn(t, func() error {
				_, err := net.Dial("tcp", table.addr)
				return err
			}, nil, 10*time.Millisecond, 100*time.Millisecond)
			conn := <-c
			assert.NotNil(t, conn)
		})
	}
}
