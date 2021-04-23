// Copyright (c) TFG Co. All Rights Reserved.
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

package packet

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPacket(t *testing.T) {
	p := New()
	assert.NotNil(t, p)
}

func TestString(t *testing.T) {
	tables := []struct {
		tp     Type
		data   []byte
		strOut string
	}{
		{Handshake, []byte{0x01}, fmt.Sprintf("Type: %d, Length: %d, Data: %s", Handshake, 1, string([]byte{0x01}))},
		{Data, []byte{0x01, 0x02, 0x03}, fmt.Sprintf("Type: %d, Length: %d, Data: %s", Data, 3, string([]byte{0x01, 0x02, 0x03}))},
		{Kick, []byte{0x05, 0x02, 0x03, 0x04}, fmt.Sprintf("Type: %d, Length: %d, Data: %s", Kick, 4, string([]byte{0x05, 0x02, 0x03, 0x04}))},
	}

	for _, table := range tables {
		t.Run(string(table.data), func(t *testing.T) {
			p := &Packet{}
			p.Data = table.data
			p.Type = table.tp
			p.Length = len(table.data)
			assert.Equal(t, table.strOut, p.String())
		})
	}
}
