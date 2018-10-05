package services

import (
	"context"

	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/examples/demo/worker/protos"
	"github.com/topfreegames/pitaya/logger"
)

// Room server
type Room struct {
	component.Base
}

// CallLog makes ReliableRPC to metagame LogRemote
func (*Room) CallLog(ctx context.Context, arg *protos.Arg) (*protos.Response, error) {
	route := "metagame.metagame.logremote"
	reply := &protos.Response{}
	jid, err := pitaya.ReliableRPC(route, nil, reply, arg)
	if err != nil {
		logger.Log.Infof("failed to enqueue rpc: %q", err)
		return nil, err
	}

	logger.Log.Infof("enqueue rpc job: %d", jid)
	return &protos.Response{Code: 200, Msg: "ok"}, nil
}
