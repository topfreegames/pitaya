package services

import (
	"context"

	"github.com/topfreegames/pitaya/v3/examples/demo/worker/protos"
	pitaya "github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/pkg/logger"
)

// Room server
type Room struct {
	component.Base
	app pitaya.Pitaya
}

// NewRoom ctor
func NewRoom(app pitaya.Pitaya) *Room {
	return &Room{app: app}
}

// CallLog makes ReliableRPC to metagame LogRemote
func (r *Room) CallLog(ctx context.Context, arg *protos.Arg) (*protos.Response, error) {
	route := "metagame.metagame.logremote"
	reply := &protos.Response{}
	jid, err := r.app.ReliableRPC(route, nil, reply, arg)
	if err != nil {
		logger.Log.Infof("failed to enqueue rpc: %q", err)
		return nil, err
	}

	logger.Log.Infof("enqueue rpc job: %d", jid)
	return &protos.Response{Code: 200, Msg: "ok"}, nil
}
