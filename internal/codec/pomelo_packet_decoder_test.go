package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/internal/packet"
)

var forwardTables = map[string]struct {
	buf []byte
	err error
}{
	"test_ok":                   {[]byte{packet.Handshake}, nil},
	"test_wrong_packet_type":    {[]byte{0x06}, packet.ErrWrongPomeloPacketType},
	"test_over_max_packet_size": {[]byte{packet.Handshake, 0xFF, 0xFF, 0xFF}, ErrPacketSizeExcced},
}

var (
	// headers
	handshakeHeader = []byte{packet.Handshake, 0x00, 0x00, 0x01}
	invalidHeader   = []byte{0x06, 0x00, 0x00, 0x01}
)

var decodeTables = map[string][]struct {
	data   []byte
	packet []*packet.Packet
	err    error
}{
	"test_not_enough_bytes":       {{[]byte{0x01}, nil, constants.ErrMessageWithNotEnoughLength}},
	"test_first_forward":          {{handshakeHeader, nil, nil}},
	"test_error_on_first_forward": {{invalidHeader, nil, packet.ErrWrongPomeloPacketType}},
	"test_second_forward": {
		{handshakeHeader, nil, nil},
		{handshakeHeader, []*packet.Packet{{packet.Handshake, 1, []byte{0x01}}}, nil},
	},
	"test_error_on_second_forward": {
		{handshakeHeader, nil, nil},
		{append([]byte{0x00}, invalidHeader...), nil, packet.ErrWrongPomeloPacketType},
	},
}

func TestNewPomeloPacketDecoder(t *testing.T) {
	t.Parallel()

	ppd := NewPomeloPacketDecoder()

	assert.NotNil(t, ppd)
}

func TestForward(t *testing.T) {
	t.Parallel()

	for name, table := range forwardTables {
		t.Run(name, func(t *testing.T) {
			ppd := NewPomeloPacketDecoder()

			ppd.buf.Write(table.buf)
			err := ppd.forward()

			assert.Equal(t, err, table.err)
		})
	}
}

func TestDecode(t *testing.T) {
	t.Parallel()

	for name, tables := range decodeTables {
		t.Run(name, func(t *testing.T) {
			ppd := NewPomeloPacketDecoder()

			for _, table := range tables {
				packet, err := ppd.Decode(table.data)

				assert.Equal(t, err, table.err)
				assert.ElementsMatch(t, packet, table.packet)
			}
		})
	}
}
