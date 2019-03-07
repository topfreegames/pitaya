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

package remote

import (
	"context"

	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/session"
)

// Sys contains logic for handling sys remotes
type Sys struct {
	component.Base
}

// BindSession binds the local session
func (s *Sys) BindSession(ctx context.Context, sessionData *protos.Session) (*protos.Response, error) {
	sess := session.GetSessionByID(sessionData.Id)
	if sess == nil {
		return nil, constants.ErrSessionNotFound
	}
	if err := sess.Bind(ctx, sessionData.Uid); err != nil {
		return nil, err
	}
	return &protos.Response{Data: []byte("ack")}, nil
}

// PushSession updates the local session
func (s *Sys) PushSession(ctx context.Context, sessionData *protos.Session) (*protos.Response, error) {
	sess := session.GetSessionByID(sessionData.Id)
	if sess == nil {
		return nil, constants.ErrSessionNotFound
	}
	if err := sess.SetDataEncoded(sessionData.Data); err != nil {
		return nil, err
	}
	return &protos.Response{Data: []byte("ack")}, nil
}

// Kick kicks a local user
func (s *Sys) Kick(ctx context.Context, msg *protos.KickMsg) (*protos.KickAnswer, error) {
	res := &protos.KickAnswer{
		Kicked: false,
	}
	sess := session.GetSessionByUID(msg.GetUserId())
	if sess == nil {
		return res, constants.ErrSessionNotFound
	}
	err := sess.Kick(ctx)
	if err != nil {
		return res, err
	}
	res.Kicked = true
	return res, nil
}
