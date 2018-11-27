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

	"github.com/topfreegames/pitaya/util/compression"
)

// Encoder interface
type Encoder interface {
	IsCompressionEnabled() bool
	Encode(message *Message) ([]byte, error)
}

// MessagesEncoder implements MessageEncoder interface
type MessagesEncoder struct {
	DataCompression bool
}

// NewMessagesEncoder returns a new message encoder
func NewMessagesEncoder(dataCompression bool) *MessagesEncoder {
	me := &MessagesEncoder{dataCompression}
	return me
}

// IsCompressionEnabled returns wether the compression is enabled or not
func (me *MessagesEncoder) IsCompressionEnabled() bool {
	return me.DataCompression
}

// Encode marshals message to binary format. Different message types is corresponding to
// different message header, message types is identified by 2-4 bit of flag field. The
// relationship between message types and message header is presented as follows:
// ------------------------------------------
// |   type   |  flag  |       other        |
// |----------|--------|--------------------|
// | request  |----000-|<message id>|<route>|
// | notify   |----001-|<route>             |
// | response |----010-|<message id>        |
// | push     |----011-|<route>             |
// ------------------------------------------
// The figure above indicates that the bit does not affect the type of message.
// See ref: https://github.com/topfreegames/pitaya/blob/master/docs/communication_protocol.md
func (me *MessagesEncoder) Encode(message *Message) ([]byte, error) {
	if invalidType(message.Type) {
		return nil, ErrWrongMessageType
	}

	buf := make([]byte, 0)
	flag := byte(message.Type) << 1

	code, compressed := routes[message.Route]
	if compressed {
		flag |= msgRouteCompressMask
	}

	if message.Err {
		flag |= errorMask
	}

	buf = append(buf, flag)

	if message.Type == Request || message.Type == Response {
		n := message.ID
		// variant length encode
		for {
			b := byte(n % 128)
			n >>= 7
			if n != 0 {
				buf = append(buf, b+128)
			} else {
				buf = append(buf, b)
				break
			}
		}
	}

	if routable(message.Type) {
		if compressed {
			buf = append(buf, byte((code>>8)&0xFF))
			buf = append(buf, byte(code&0xFF))
		} else {
			buf = append(buf, byte(len(message.Route)))
			buf = append(buf, []byte(message.Route)...)
		}
	}

	if me.DataCompression {
		d, err := compression.DeflateData(message.Data)
		if err != nil {
			return nil, err
		}

		if len(d) < len(message.Data) {
			message.Data = d
			buf[0] |= gzipMask
		}
	}

	buf = append(buf, message.Data...)
	return buf, nil
}

// Decode decodes the message
func (me *MessagesEncoder) Decode(data []byte) (*Message, error) {
	return Decode(data)
}

// Decode unmarshal the bytes slice to a message
// See ref: https://github.com/topfreegames/pitaya/blob/master/docs/communication_protocol.md
func Decode(data []byte) (*Message, error) {
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
			m.compressed = true
			code := binary.BigEndian.Uint16(data[offset:(offset + 2)])
			route, ok := codes[code]
			if !ok {
				return nil, ErrRouteInfoNotFound
			}
			m.Route = route
			offset += 2
		} else {
			m.compressed = false
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
