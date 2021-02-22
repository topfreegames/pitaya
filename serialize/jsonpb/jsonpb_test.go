package jsonpb

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v2/constants"
	"github.com/topfreegames/pitaya/v2/protos"
	"github.com/topfreegames/pitaya/v2/serialize/json"
)

func TestMarshal(t *testing.T) {
	type JSONStruct struct {
		FieldA string `json:"field_a"`
	}

	tests := map[string]struct {
		raw     interface{}
		errType interface{}
	}{
		"ValidProtobufMessageSuccessMarshal": {&protos.Request{Type: protos.RPCType_User, FrontendID: "abc"}, nil},
		"ValidJSONStructFailsMarshal":        {JSONStruct{FieldA: "test"}, constants.ErrWrongValueType},
		"InvalidProtobufMessageFailsMarshal": {"invalid", constants.ErrWrongValueType},
	}

	serializer := NewSerializer()
	jsonSerializer := json.NewSerializer()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			res, err := serializer.Marshal(test.raw)
			assert.Equal(t, test.errType, err)

			if test.errType == nil {
				jsonRes, jsonErr := jsonSerializer.Marshal(test.raw)
				assert.NoError(t, jsonErr)

				// remove spaces
				bytes.ReplaceAll(jsonRes, []byte(" "), []byte{})
				bytes.ReplaceAll(res, []byte(" "), []byte{})
				assert.Equal(t, jsonRes, res)
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	type JSONStruct struct {
		FieldA string `json:"field_a"`
	}

	tests := map[string]struct {
		expected  proto.Message
		dest      proto.Message
		marshaled []byte
		errType   interface{}
	}{
		"ValidJSONMessageSuccessUnmarshal": {&protos.Request{Type: protos.RPCType_User, FrontendID: "abc", Metadata: []byte{}}, &protos.Request{}, []byte("{\"type\":1,\"frontendID\":\"abc\"}"), nil},
		"InvalidMessageFailsMarshal":       {nil, &protos.Request{}, []byte(""), constants.ErrWrongValueType},
	}

	serializer := NewSerializer()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := serializer.Unmarshal(test.marshaled, test.dest)
			assert.Equal(t, test.errType, err)
			if test.errType == nil {
				assert.True(t, proto.Equal(test.expected, test.dest))
			}
		})
	}
}
