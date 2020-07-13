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

package modules

import (
	"context"

	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/session"
)

// UniqueSession module watches for sessions using the same UID and kicks them
type UniqueSession struct {
	Base
	server      *cluster.Server
	rpcClient   cluster.RPCClient
	sessionPool session.SessionPool
}

// NewUniqueSession creates a new unique session module
func NewUniqueSession(server *cluster.Server, rpcServer cluster.RPCServer, rpcClient cluster.RPCClient, sessionPool session.SessionPool) *UniqueSession {
	return &UniqueSession{
		server:      server,
		rpcClient:   rpcClient,
		sessionPool: sessionPool,
	}
}

// OnUserBind method should be called when a user binds a session in remote servers
func (u *UniqueSession) OnUserBind(uid, fid string) {
	if u.server.ID == fid {
		return
	}
	oldSession := u.sessionPool.GetSessionByUID(uid)
	if oldSession != nil {
		// TODO: it would be nice to set this correctly
		oldSession.Kick(context.Background())
	}
}

// Init initializes the module
func (u *UniqueSession) Init() error {
	u.sessionPool.OnSessionBind(func(ctx context.Context, s session.Session) error {
		oldSession := u.sessionPool.GetSessionByUID(s.UID())
		if oldSession != nil {
			return oldSession.Kick(ctx)
		}
		err := u.rpcClient.BroadcastSessionBind(s.UID())
		return err
	})
	return nil
}
