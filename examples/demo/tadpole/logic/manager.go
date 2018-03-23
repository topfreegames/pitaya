package logic

import (
	"log"
	"strconv"

	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/examples/demo/tadpole/logic/protocol"
	"github.com/topfreegames/pitaya/session"
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
func (m *Manager) Login(s *session.Session, msg *protocol.JoyLoginRequest) (protocol.LoginResponse, error) {
	log.Println(msg)
	id := s.ID()
	s.Bind(strconv.Itoa(int(id)))
	resp := protocol.LoginResponse{
		Status: protocol.LoginStatusSucc,
		ID:     id,
	}
	return resp, nil
}
