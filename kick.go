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

package pitaya

import (
	"context"

	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/session"
)

// SendKickToUsers sends kick to an user array
func SendKickToUsers(uids []string, frontendType string) ([]string, error) {
	if !app.server.Frontend && frontendType == "" {
		return uids, constants.ErrFrontendTypeNotSpecified
	}

	var notKickedUids []string

	for _, uid := range uids {
		if s := session.GetSessionByUID(uid); s != nil {
			if err := s.Kick(context.Background()); err != nil {
				notKickedUids = append(notKickedUids, uid)
				logger.Log.Errorf("Session kick error, ID=%d, UID=%s, ERROR=%s", s.ID(), s.UID(), err.Error())
			}
		} else if app.rpcClient != nil {
			kick := &protos.KickMsg{UserId: uid}
			if err := app.rpcClient.SendKick(uid, frontendType, kick); err != nil {
				notKickedUids = append(notKickedUids, uid)
				logger.Log.Errorf("RPCClient send kick error, UID=%s, SvType=%s, Error=%s", uid, frontendType, err.Error())
			}
		} else {
			notKickedUids = append(notKickedUids, uid)
		}

	}

	if len(notKickedUids) != 0 {
		return notKickedUids, constants.ErrKickingUsers
	}

	return nil, nil
}
