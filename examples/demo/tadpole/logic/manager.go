package logic

import (
	"context"
	"log"
	"strconv"

	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/examples/demo/tadpole/logic/protocol"
)

// Manager component
type Manager struct {
	component.Base
}

// NewManager returns  a new manager instance
func NewManager() *Manager {
	return &Manager{}
}

// Login handler was used to guest login
func (m *Manager) Login(ctx context.Context, msg *protocol.JoyLoginRequest) (protocol.LoginResponse, error) {
	log.Println(msg)
	s := pitaya.GetSessionFromCtx(ctx)
	id := s.ID()
	s.Bind(ctx, strconv.Itoa(int(id)))
	resp := protocol.LoginResponse{
		Status: protocol.LoginStatusSucc,
		ID:     id,
	}
	return resp, nil
}
