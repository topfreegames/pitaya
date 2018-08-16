package pitaya

import (
	"context"

	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/session"
)

// SendKickToUser sends kick to an user
func SendKickToUser(uid, frontendType string) error {
	if s := session.GetSessionByUID(uid); s != nil {
		if err := s.Kick(context.Background()); err != nil {
			logger.Log.Errorf("Session kick error, ID=%d, UID=%d, ERROR=%s", s.ID(), s.UID(), err.Error())
		}
	} else if app.rpcClient != nil {
		kick := &protos.KickMsg{UserId: uid}
		if err := app.rpcClient.SendKick(uid, frontendType, kick); err != nil {
			logger.Log.Errorf("RPCClient send kick error, UID=%d, SvType=%s, Error=%s", uid, frontendType, err.Error())
		}
	}
	return nil
}
