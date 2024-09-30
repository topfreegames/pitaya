package util

import (
	e "github.com/topfreegames/pitaya/v3/pkg/errors"
	"github.com/topfreegames/pitaya/v3/pkg/protos"
	"github.com/topfreegames/pitaya/v3/pkg/serialize"
	"github.com/topfreegames/pitaya/v3/pkg/serialize/json"
	"github.com/topfreegames/pitaya/v3/pkg/serialize/protobuf"
)

var (
	DefaultErrWrapper serialize.ErrorWrapper = &pitayaErrWrapper{}
)

type pitayaErrWrapper struct {
}

func (p pitayaErrWrapper) Marshal(err error, serializer serialize.Serializer) ([]byte, error) {
	code := e.ErrUnknownCode
	msg := err.Error()
	metadata := map[string]string{}
	if val, ok := err.(*e.Error); ok {
		code = val.Code
		metadata = val.Metadata
	}
	errPayload := &protos.Error{
		Code: code,
		Msg:  msg,
	}
	if len(metadata) > 0 {
		errPayload.Metadata = metadata
	}
	return SerializeOrRaw(serializer, errPayload)
}

func (p pitayaErrWrapper) Unmarshal(payload []byte, serializer serialize.Serializer) *e.Error {
	err := &e.Error{Code: e.ErrUnknownCode}
	switch serializer.(type) {
	case *json.Serializer:
		_ = serializer.Unmarshal(payload, err)
	case *protobuf.Serializer:
		pErr := &protos.Error{Code: e.ErrUnknownCode}
		_ = serializer.Unmarshal(payload, pErr)
		err = &e.Error{Code: pErr.Code, Message: pErr.Msg, Metadata: pErr.Metadata}
	}
	return err
}
