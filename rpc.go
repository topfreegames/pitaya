// Copyright (c) nano Author and TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package pitaya

import (
	"context"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/worker"
)

// RPC calls a method in a different server
func RPC(ctx context.Context, routeStr string, reply proto.Message, arg proto.Message) error {
	return doSendRPC(ctx, "", routeStr, reply, arg)
}

// RPCTo send a rpc to a specific server
func RPCTo(ctx context.Context, serverID, routeStr string, reply proto.Message, arg proto.Message) error {
	return doSendRPC(ctx, serverID, routeStr, reply, arg)
}

// ReliableRPC enqueues RPC to worker so it's executed asynchronously
// Default enqueue options are used
func ReliableRPC(
	routeStr string,
	metadata map[string]interface{},
	reply, arg proto.Message,
) (jid string, err error) {
	return app.worker.EnqueueRPC(routeStr, metadata, reply, arg)
}

// ReliableRPCWithOptions enqueues RPC to worker
// Receive worker options for this specific RPC
func ReliableRPCWithOptions(
	routeStr string,
	metadata map[string]interface{},
	reply, arg proto.Message,
	opts *worker.EnqueueOpts,
) (jid string, err error) {
	return app.worker.EnqueueRPCWithOptions(routeStr, metadata, reply, arg, opts)
}

func doSendRPC(ctx context.Context, serverID, routeStr string, reply proto.Message, arg proto.Message) error {
	if app.rpcServer == nil {
		return constants.ErrRPCServerNotInitialized
	}

	if reflect.TypeOf(reply).Kind() != reflect.Ptr {
		return constants.ErrReplyShouldBePtr
	}

	r, err := route.Decode(routeStr)
	if err != nil {
		return err
	}

	if r.SvType == "" {
		return constants.ErrNoServerTypeChosenForRPC
	}

	if (r.SvType == app.server.Type && serverID == "") || serverID == app.server.ID {
		return constants.ErrNonsenseRPC
	}

	return remoteService.RPC(ctx, serverID, r, reply, arg)
}
