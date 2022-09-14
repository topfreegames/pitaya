package sidecar

import (
	"context"
	"github.com/topfreegames/pitaya/v2/pkg/protos"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync"
)

// Call struct represents an incoming RPC call from other servers
type Call struct {
	ctx   context.Context
	req   *protos.Request
	done  chan (bool)
	res   *protos.Response
	err   *protos.Error
	reqId uint64
}

var (
	stopChan = make(chan bool)
	callChan = make(chan *Call)
	sdChan = make(chan *protos.SDEvent)
	shouldRun = true
	redId uint64
	reqMutex sync.RWMutex
	reqMap = make(map[uint64]*Call)
	empty = &emptypb.Empty{}
)
