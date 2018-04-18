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

package codec

import (
	"bytes"

	"github.com/topfreegames/pitaya/internal/packet"
)

// PomeloPacketDecoder reads and decodes network data slice following pomelo's protocol
type PomeloPacketDecoder struct{}

// NewPomeloPacketDecoder returns a new decoder that used for decode network bytes slice.
func NewPomeloPacketDecoder() *PomeloPacketDecoder {
	return &PomeloPacketDecoder{}
}

func (c *PomeloPacketDecoder) forward(buf *bytes.Buffer) (int, packet.Type, error) {
	header := buf.Next(HeadLength)
	typ := header[0]
	if typ < packet.Handshake || typ > packet.Kick {
		return 0, 0x00, packet.ErrWrongPomeloPacketType
	}
	size := bytesToInt(header[1:])

	// packet length limitation
	if size > MaxPacketSize {
		return 0, 0x00, ErrPacketSizeExcced
	}
	return size, packet.Type(typ), nil
}

// Decode decode the network bytes slice to packet.Packet(s)
func (c *PomeloPacketDecoder) Decode(data []byte) ([]*packet.Packet, error) {
	buf := bytes.NewBuffer(nil)
	buf.Write(data)

	var (
		packets []*packet.Packet
		err     error
	)
	// check length
	if buf.Len() < HeadLength {
		return nil, nil
	}

	// first time
	size, typ, err := c.forward(buf)
	if err != nil {
		return nil, err
	}

	for size <= buf.Len() {
		p := &packet.Packet{Type: typ, Length: size, Data: buf.Next(size)}
		packets = append(packets, p)

		// more packet
		if buf.Len() < HeadLength {
			break
		}

		size, typ, err = c.forward(buf)
		if err != nil {
			return nil, err
		}
	}

	return packets, nil
}

// Decode packet data length byte to int(Big end)
func bytesToInt(b []byte) int {
	result := 0
	for _, v := range b {
		result = result<<8 + int(v)
	}
	return result
}
