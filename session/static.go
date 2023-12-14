package session

import (
	"context"
	"github.com/topfreegames/pitaya/v2/networkentity"
)

var DefaultSessionPool SessionPool

// GetSessionByUID return a session bound to an user id
func GetSessionByUID(uid string) Session {
	return DefaultSessionPool.GetSessionByUID(uid)
}

// GetSessionByID return a session bound to a frontend server id
func GetSessionByID(id int64) Session {
	return DefaultSessionPool.GetSessionByID(id)
}

// OnSessionBind adds a method to be called when a session is bound
// same function cannot be added twice!
func OnSessionBind(f func(ctx context.Context, s Session) error) {
	DefaultSessionPool.OnSessionBind(f)
}

// OnAfterSessionBind adds a method to be called when session is bound and after all sessionBind callbacks
func OnAfterSessionBind(f func(ctx context.Context, s Session) error) {
	DefaultSessionPool.OnAfterSessionBind(f)
}

// OnSessionClose adds a method that will be called when every session closes
func OnSessionClose(f func(s Session)) {
	DefaultSessionPool.OnSessionClose(f)
}

// CloseAll calls Close on all sessions
func CloseAll() {
	DefaultSessionPool.CloseAll()
}

// New returns a new session instance
// a NetworkEntity is a low-level network instance
func New(entity networkentity.NetworkEntity, frontend bool, UID ...string) Session {
	DefaultSessionPool = NewSessionPool()
	userID := ""
	if len(UID) > 0 {
		userID = UID[0]
	}

	session := DefaultSessionPool.NewSession(entity, frontend, userID)
	return session
}
