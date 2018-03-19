// +build benchmark

package io

import (
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/benchmark/testdata"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/serialize/protobuf"
	"github.com/topfreegames/pitaya/session"
)

const (
	addr = "127.0.0.1:13250" // local address
	conc = 1000              // concurrent client count
)

//
type TestHandler struct {
	component.Base
	metrics int32
}

func (h *TestHandler) AfterInit() {
	ticker := time.NewTicker(time.Second)

	// metrics output ticker
	go func() {
		for range ticker.C {
			println("QPS", atomic.LoadInt32(&h.metrics))
			atomic.StoreInt32(&h.metrics, 0)
		}
	}()
}

func (h *TestHandler) Ping(s *session.Session, data *testdata.Ping) error {
	atomic.AddInt32(&h.metrics, 1)
	return s.Push("pong", &testdata.Pong{Content: data.Content})
}

func server() {
	pitaya.Register(&TestHandler{})
	pitaya.SetSerializer(protobuf.NewSerializer())

	pitaya.Listen(addr)
}

func client() {
	c := NewConnector()

	if err := c.Start(addr); err != nil {
		panic(err)
	}

	chReady := make(chan struct{})
	c.OnConnected(func() {
		chReady <- struct{}{}
	})

	c.On("pong", func(data interface{}) {})

	<-chReady
	for {
		c.Notify("TestHandler.Ping", &testdata.Ping{})
		time.Sleep(10 * time.Millisecond)
	}
}

func TestIO(t *testing.T) {
	go server()

	// wait server startup
	time.Sleep(1 * time.Second)
	for i := 0; i < conc; i++ {
		go client()
	}

	log.SetFlags(log.LstdFlags | log.Llongfile)

	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)

	<-sg

	t.Log("exit")
}
