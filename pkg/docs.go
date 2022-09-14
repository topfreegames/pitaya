package pkg

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/topfreegames/pitaya/v2/pkg/component"
	pitayaprotos "github.com/topfreegames/pitaya/v2/pkg/protos"
)

// Docs is the handler that will serve protos
type Docs struct {
	component.Base
}

// NewDocs creates a new docs handler
func NewDocs() *Docs {
	return &Docs{}
}

// Docs returns documentation
func (c *Docs) Docs(ctx context.Context) (*pitayaprotos.Doc, error) {
	d, err := Documentation(true)
	if err != nil {
		return nil, err
	}
	doc, err := json.Marshal(d)

	if err != nil {
		return nil, err
	}

	return &pitayaprotos.Doc{Doc: string(doc)}, nil
}

// Protos return protobuffers descriptors
func (c *Docs) Protos(ctx context.Context, message *pitayaprotos.ProtoNames) (*pitayaprotos.ProtoDescriptors, error) {
	descriptors := make([][]byte, 0, len(message.GetName()))

	for _, name := range message.GetName() {
		if !strings.HasPrefix(name, "protos.") {
			name = strings.Replace(name, strings.SplitN(name, ".", 2)[0], "google.protobuf", 1)
		}
		protoDescriptor, err := Descriptor(name)
		if err != nil {
			return nil, err
		}
		descriptors = append(descriptors, protoDescriptor)
	}

	return &pitayaprotos.ProtoDescriptors{
		Desc: descriptors,
	}, nil
}
