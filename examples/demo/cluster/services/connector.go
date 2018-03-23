package services

import (
	"fmt"

	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/session"
)

// ConnectorRemote is a remote that will receive rpc's
type ConnectorRemote struct {
	component.Base
}

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

// RemoteFunc is a function that will be called remotelly
func (c *ConnectorRemote) RemoteFunc(message string) (*RPCResponse, error) {
	fmt.Printf("received a remote call with this message: %s\n", message)
	return &RPCResponse{
		Msg: message,
	}, nil
}
