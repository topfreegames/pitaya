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

package message

import (
	"encoding/binary"
	"github.com/topfreegames/pitaya/v3/pkg/util/compression"
)

// Decoder interface
type Decoder interface {
	IsCompressionEnabled() bool
	Decode(data []byte) (*Message, error)
}

// MessagesDecoder implements MessageEncoder interface
type MessagesDecoder struct {
	DataCompression bool
}

// NewMessagesDecoder returns a new message decoder
func NewMessagesDecoder(dataCompression bool) *MessagesDecoder {
	me := &MessagesDecoder{dataCompression}
	return me
}

// IsCompressionEnabled returns wether the compression is enabled or not
func (md *MessagesDecoder) IsCompressionEnabled() bool {
	return md.DataCompression
}

func (md *MessagesDecoder) Decode(data []byte) (*Message, error) {
	if len(data) < msgHeadLength {
		return nil, ErrInvalidMessage
	}
	m := New()
	flag := data[0]
	offset := 1
	m.Type = Type((flag >> 1) & msgTypeMask)

	if invalidType(m.Type) {
		return nil, ErrWrongMessageType
	}

	if m.Type == Request || m.Type == Response {
		id := uint(0)
		// little end byte order
		// WARNING: must can be stored in 64 bits integer
		// variant length encode
		for i := offset; i < len(data); i++ {
			b := data[i]
			id += uint(b&0x7F) << uint(7*(i-offset))
			if b < 128 {
				offset = i + 1
				break
			}
		}
		m.ID = id
	}

	m.Err = flag&errorMask == errorMask

	if routable(m.Type) {
		if flag&msgRouteCompressMask == 1 {
			m.Compressed = true
			code := binary.BigEndian.Uint16(data[offset:(offset + 2)])
			routesCodesMutex.RLock()
			route, ok := codes[code]
			routesCodesMutex.RUnlock()
			if !ok {
				return nil, ErrRouteInfoNotFound
			}
			m.Route = route
			offset += 2
		} else {
			m.Compressed = false
			rl := data[offset]
			offset++
			m.Route = string(data[offset:(offset + int(rl))])
			offset += int(rl)
		}
	}

	m.Data = data[offset:]
	var err error
	if flag&gzipMask == gzipMask {
		m.Data, err = compression.InflateData(m.Data)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}
