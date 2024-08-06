package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/topfreegames/pitaya/v3/examples/demo/cluster_protos"
	pitaya "github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	pitayaprotos "github.com/topfreegames/pitaya/v3/pkg/protos"
)

// ConnectorRemote is a remote that will receive rpc's
type ConnectorRemote struct {
	component.Base
	app pitaya.Pitaya
}

// Connector struct
type Connector struct {
	component.Base
	app pitaya.Pitaya
}

// SessionData struct
type SessionData struct {
	Data map[string]interface{}
}

// Response struct
type Response struct {
	Code int32
	Msg  string
}

// NewConnector ctor
func NewConnector(app pitaya.Pitaya) *Connector {
	return &Connector{app: app}
}

// NewConnectorRemote ctor
func NewConnectorRemote(app pitaya.Pitaya) *ConnectorRemote {
	return &ConnectorRemote{app: app}
}

func reply(code int32, msg string) (*cluster_protos.Response, error) {
	res := &cluster_protos.Response{
		Code: code,
		Msg:  msg,
	}
	return res, nil
}

// GetSessionData gets the session data
func (c *Connector) GetSessionData(ctx context.Context) (*SessionData, error) {
	s := c.app.GetSessionFromCtx(ctx)
	res := &SessionData{
		Data: s.GetData(),
	}
	return res, nil
}

// SetSessionData sets the session data
func (c *Connector) SetSessionData(ctx context.Context, data *SessionData) (*cluster_protos.Response, error) {
	s := c.app.GetSessionFromCtx(ctx)
	err := s.SetData(data.Data)
	if err != nil {
		return nil, pitaya.Error(err, "CN-000", map[string]string{"failed": "set data"})
	}
	return reply(200, "success")
}

// NotifySessionData sets the session data
func (c *Connector) NotifySessionData(ctx context.Context, data *SessionData) {
	s := c.app.GetSessionFromCtx(ctx)
	err := s.SetData(data.Data)
	if err != nil {
		fmt.Println("got error on notify", err)
	}
}

// RemoteFunc is a function that will be called remotely
func (c *ConnectorRemote) RemoteFunc(ctx context.Context, msg *cluster_protos.RPCMsg) (*cluster_protos.RPCRes, error) {
	fmt.Printf("received a remote call with this message: %s\n", msg.GetMsg())
	return &cluster_protos.RPCRes{
		Msg: msg.GetMsg(),
	}, nil
}

// Docs returns documentation
func (c *Connector) Docs(ctx context.Context, ddd *pitayaprotos.Doc) (*pitayaprotos.Doc, error) {
	d, err := c.app.Documentation(true)
	if err != nil {
		return nil, err
	}
	doc, err := json.Marshal(d)

	if err != nil {
		return nil, err
	}

	return &pitayaprotos.Doc{Doc: string(doc)}, nil
}

func (c *Connector) Descriptor(ctx context.Context, names *pitayaprotos.ProtoNames) (*pitayaprotos.ProtoDescriptors, error) {
	descriptors := make([][]byte, 0)

	for _, protoName := range names.Name {
		desc, err := pitaya.Descriptor(protoName)
		if err != nil {
			fmt.Println(fmt.Errorf("failed to get descriptor for '%s': %w", protoName, err))
			continue
		}

		descriptors = append(descriptors, desc)
	}

	return &pitayaprotos.ProtoDescriptors{Desc: descriptors}, nil
}
