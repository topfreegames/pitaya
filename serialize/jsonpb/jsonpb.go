package jsonpb

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/topfreegames/pitaya/v2/constants"
)

// Serializer implements the serialize.Serializer interface
type Serializer struct {
	marshaler *jsonpb.Marshaler
}

// NewSerializer returns a new Serializer.
func NewSerializer() *Serializer {
	return &Serializer{
		marshaler: &jsonpb.Marshaler{EnumsAsInts: true},
	}
}

// Marshal returns the JSON encoding of v.
func (s *Serializer) Marshal(v interface{}) ([]byte, error) {
	pb, ok := v.(proto.Message)
	if !ok {
		return []byte{}, constants.ErrWrongValueType
	}

	res, err := s.marshaler.MarshalToString(pb)
	if err != nil {
		return []byte{}, constants.ErrWrongValueType
	}

	return []byte(res), nil
}

// Unmarshal parses the JSON-encoded data and stores the result
// in the value pointed to by v.
func (s *Serializer) Unmarshal(data []byte, v interface{}) error {
	pb, ok := v.(proto.Message)
	if !ok {
		return constants.ErrWrongValueType
	}

	err := jsonpb.UnmarshalString(string(data), pb)
	if err != nil {
		return constants.ErrWrongValueType
	}

	return nil
}

// GetName returns the name of the serializer.
func (s *Serializer) GetName() string {
	// NOTE: this serializer doesn't use a different name than `json` because it
	// must behave exactly the same as `json` on the client side.
	return "json"
}
