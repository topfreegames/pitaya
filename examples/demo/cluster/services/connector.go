package services

import (
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/session"
)

// Connector struct
type Connector struct {
	component.Base
}

// SessionData is the session data struct
type SessionData struct {
	Data map[string]interface{} `json:"data"`
}

// GetSessionData gets the session data
func (c *Connector) GetSessionData(s *session.Session, data *SessionData) error {
	return s.Response(s.GetData())
}

// SetSessionData sets the session data
func (c *Connector) SetSessionData(s *session.Session, data *SessionData) error {
	err := s.SetData(data.Data)
	if err != nil {
		return err
	}
	return s.Response("success")
}
