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

// SessionData struct
type SessionData struct {
	Data map[string]interface{}
}

// GetSessionData gets the session data
func (c *Connector) GetSessionData(s *session.Session) (map[string]interface{}, error) {
	return s.GetData(), nil
}

// SetSessionData sets the session data
func (c *Connector) SetSessionData(s *session.Session, data *SessionData) (string, error) {
	err := s.SetData(data.Data)
	if err != nil {
		return "", err
	}
	return "success", nil
}

// NotifySessionData sets the session data
func (c *Connector) NotifySessionData(s *session.Session, data *SessionData) {
	err := s.SetData(data.Data)
	if err != nil {
		fmt.Println("got error on notify", err)
	}
}

// RemoteFunc is a function that will be called remotely
func (c *ConnectorRemote) RemoteFunc(message string) (*RPCResponse, error) {
	fmt.Printf("received a remote call with this message: %s\n", message)
	return &RPCResponse{
		Msg: message,
	}, nil
}
