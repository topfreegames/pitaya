package main

import (
	"context"
	"flag"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/topfreegames/pitaya/examples/demo/protos"
	pitaya "github.com/topfreegames/pitaya/pkg"
	"github.com/topfreegames/pitaya/pkg/logger"
	"github.com/topfreegames/pitaya/pkg/serialize/protobuf"
)

var shouldRun = true
var route *string
var answers uint64

type worker struct {
	ticker *time.Ticker
	id     int
}

func (w *worker) doSendRPCs() {
	logger.Log.Infof("starting worker %d", w.id)
	for shouldRun {
		select {
		case <-w.ticker.C:
			res := &protos.RPCMsg{}
			err := pitaya.RPC(context.Background(), *route, res, &protos.RPCMsg{})
			if err != nil {
				fmt.Printf("Error sending RPC: %s", err.Error())
			}
			logger.Log.Debugf("got response: %s", res.Msg)
			atomic.AddUint64(&answers, 1)
		}
	}
}

func main() {
	svType := flag.String("type", "bench", "the server type")
	isFrontend := flag.Bool("frontend", false, "if server is frontend")
	debug := flag.Bool("debug", false, "should log debug messages")
	route = flag.String("route", "csharp.testremote.test", "the route to stress")
	threads := flag.Int("threads", 4, "number of threads to use when sending RPCs")
	msgSec := flag.Int("rate", 1000, "messages per second rate that each thread should send")

	flag.Parse()

	defer pitaya.Shutdown()

	ser := protobuf.NewSerializer()

	l := logrus.New()
	l.SetLevel(logrus.InfoLevel)
	if *debug {
		l.SetLevel(logrus.DebugLevel)
	}
	logger.Log = l

	pitaya.SetSerializer(ser)

	pitaya.Configure(*isFrontend, *svType, pitaya.Cluster, map[string]string{})
	dieChan := pitaya.GetDieChan()
	go pitaya.Start()

	// Wait for pitaya to run, maybe we can improve this
	time.Sleep(1 * time.Second)
	tickerResetAnswer := time.NewTicker(1 * time.Second)

	workers := make([]*worker, *threads)

	for i := 0; i < *threads; i++ {
		workers[i] = &worker{
			ticker: time.NewTicker(
				time.Duration(1000000 / *msgSec) * time.Microsecond),
			id: i,
		}
		go workers[i].doSendRPCs()
	}

	for shouldRun {
		select {
		case <-tickerResetAnswer.C:
			logger.Log.Infof("Current rate: %d msg/sec", answers)
			atomic.StoreUint64(&answers, 0)
		case <-dieChan:
			logger.Log.Info("pitaya is sent die signal, killing bench")
			shouldRun = false
		}
	}
}
