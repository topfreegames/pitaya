package services

import (
	"context"

	"google.golang.org/protobuf/proto"
	"github.com/topfreegames/pitaya/v3/examples/demo/worker/protos"
	pitaya "github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/component"
)

// Worker server
type Worker struct {
	component.Base
}

// Configure starts workers and register rpc job
func (w *Worker) Configure(app pitaya.Pitaya) {
	app.StartWorker()
	app.RegisterRPCJob(&RPCJob{app: app})
}

// RPCJob implements worker.RPCJob
type RPCJob struct {
	app pitaya.Pitaya
}

// ServerDiscovery returns a serverID="", meaning any server
// is ok
func (r *RPCJob) ServerDiscovery(
	route string,
	rpcMetadata map[string]interface{},
) (serverID string, err error) {
	return "", nil
}

// RPC calls pitaya's rpc
func (r *RPCJob) RPC(
	ctx context.Context,
	serverID, routeStr string,
	reply, arg proto.Message,
) error {
	return r.app.RPCTo(ctx, serverID, routeStr, reply, arg)
}

// GetArgReply returns reply and arg of LogRemote,
// since we have no other methods in this example
func (r *RPCJob) GetArgReply(
	route string,
) (arg, reply proto.Message, err error) {
	return &protos.Arg{}, &protos.Response{}, nil
}
