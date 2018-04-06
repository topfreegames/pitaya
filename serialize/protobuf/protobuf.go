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

package protobuf

import (
	"io"
	"io/ioutil"

	"github.com/gogo/protobuf/proto"
	"github.com/topfreegames/pitaya/constants"
)

// Serializer implements the serialize.Serializer interface
type Serializer struct {
	Protos        string
	ProtosMapping string
}

// NewSerializer returns a new Serializer.
// ProtosMapping is a mapping between route and proto to be used for in and out
// e.g:
//	{
//		connector.getsessiondata: {
//			server: SessionData
//		},
//		connector.setsessiondata: {
//			client: SessionData,
//			server: SetSessionDataAnswer
//		}
//	}
func NewSerializer(protos io.Reader, protosMapping io.Reader) (*Serializer, error) {
	b, err := ioutil.ReadAll(protos)
	if err != nil {
		return nil, err
	}
	bm, err := ioutil.ReadAll(protosMapping)
	if err != nil {
		return nil, err
	}
	return &Serializer{
		Protos:        string(b),
		ProtosMapping: string(bm),
	}, nil
}

// Marshal returns the protobuf encoding of v.
func (s *Serializer) Marshal(v interface{}) ([]byte, error) {
	pb, ok := v.(proto.Message)
	if !ok {
		return nil, constants.ErrWrongValueType
	}
	return proto.Marshal(pb)
}

// Unmarshal parses the protobuf-encoded data and stores the result
// in the value pointed to by v.
func (s *Serializer) Unmarshal(data []byte, v interface{}) error {
	pb, ok := v.(proto.Message)
	if !ok {
		return constants.ErrWrongValueType
	}
	return proto.Unmarshal(data, pb)
}
