package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/examples/demo/protos"
	pitayaprotos "github.com/topfreegames/pitaya/protos"
)

// ConnectorRemote is a remote that will receive rpc's
type ConnectorRemote struct {
	component.Base
}

// Connector struct
type Connector struct {
	component.Base
}

func reply(code int32, msg string) (*protos.Response, error) {
	res := &protos.Response{
		Code: code,
		Msg:  msg,
	}
	return res, nil
}

// RemoteFunc is a function that will be called remotely
func (c *ConnectorRemote) RemoteFunc(ctx context.Context, msg *protos.RPCMsg) (*protos.RPCRes, error) {
	fmt.Printf("received a remote call with this message: %s\n", msg.GetMsg())
	return &protos.RPCRes{
		Msg: msg.GetMsg(),
	}, nil
}

// Docs returns documentation
func (c *ConnectorRemote) Docs(ctx context.Context) (*pitayaprotos.Doc, error) {
	d, err := pitaya.Documentation(true)
	if err != nil {
		return nil, err
	}
	doc, err := json.Marshal(d)

	if err != nil {
		return nil, err
	}

	fmt.Println(string(doc))
	return &pitayaprotos.Doc{Doc: string(doc)}, nil
}

func (c *ConnectorRemote) Descriptor(ctx context.Context, names *pitayaprotos.ProtoNames) (*pitayaprotos.ProtoDescriptors, error) {
	descriptors := make([][]byte, len(names.Name))

	for i, protoName := range names.Name {
		desc, err := pitaya.Descriptor(protoName)
		if err != nil {
			return nil, fmt.Errorf("failed to get descriptor for '%s': %w", protoName, err)
		}

		descriptors[i] = desc
	}

	return &pitayaprotos.ProtoDescriptors{Desc: descriptors}, nil
}
