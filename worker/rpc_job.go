// Copyright (c) TFG Co. All Rights Reserved.
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

package worker

import (
	"context"

	"github.com/golang/protobuf/proto"
)

// RPCJob has infos to execute a rpc on worker
type RPCJob interface {
	// ServerDiscovery returns a serverID based on the route
	// and any metadata that is necessary to decide
	ServerDiscovery(
		route string,
		rpcMetadata map[string]interface{},
	) (serverID string, err error)

	// RPC executes the RPC
	// It is expected that if serverID is "" the RPC
	// happens to any destiny server
	RPC(
		ctx context.Context,
		serverID, routeStr string,
		reply, arg proto.Message,
	) error

	// GetArgReply returns the arg and reply of the
	// method
	GetArgReply(route string) (arg, reply proto.Message, err error)
}
