// Copyright (c) nano Author and TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package serialize

import (
	"errors"

	"github.com/topfreegames/pitaya/v2/serialize/json"
	"github.com/topfreegames/pitaya/v2/serialize/protobuf"
)

const (
	JSON     Type = 1
	PROTOBUF Type = 2
)

type (
	// Type is the Serializer type.
	Type uint16

	// Marshaler represents a marshal interface
	Marshaler interface {
		Marshal(interface{}) ([]byte, error)
	}

	// Unmarshaler represents a Unmarshal interface
	Unmarshaler interface {
		Unmarshal([]byte, interface{}) error
	}

	// Serializer is the interface that groups the basic Marshal and Unmarshal methods.
	Serializer interface {
		Marshaler
		Unmarshaler
		GetName() string
	}
)

// All recognized and expected serializer type values.

// NewSerializer returns a new serializer of the respective type (JSON or PROTOBUF) according to serializerType Type.
// If serializerType is a JSON, then a JSON serializer is returned.
// If serializerType is a PROTOBUF, then  a PROTOBUF serializer is returned.
// Otherwise, if serializerType is not a valid serializer type, then it returns nil.
func NewSerializer(serializerType Type) (Serializer, error) { //nolint:ireturn
	switch serializerType {
	case JSON:
		return json.NewSerializer(), nil
	case PROTOBUF:
		return protobuf.NewSerializer(), nil
	default:
		return nil, errors.New("serializer type unknown")
	}
}
