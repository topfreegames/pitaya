package services

import (
	"context"
	"fmt"

	demoProtos "github.com/topfreegames/pitaya/examples/demo/protos"
	"github.com/topfreegames/pitaya/pkg/component"
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

// RemoteFunc is a function that will be called remotelly
func (c *ConnectorRemote) RemoteFunc(ctx context.Context, msg *demoProtos.RPCMsg) (*demoProtos.RPCMsg, error) {
	fmt.Printf("received a remote call with this message: %s\n", msg.Msg)
	return msg, nil
}
