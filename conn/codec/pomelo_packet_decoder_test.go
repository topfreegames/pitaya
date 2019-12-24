package codec

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/conn/packet"
)

var forwardTables = map[string]struct {
	buf []byte
	err error
}{
	"test_handshake_type":     {[]byte{packet.Handshake, 0x00, 0x00, 0x00}, nil},
	"test_handshake_ack_type": {[]byte{packet.HandshakeAck, 0x00, 0x00, 0x00}, nil},
	"test_heartbeat_type":     {[]byte{packet.Heartbeat, 0x00, 0x00, 0x00}, nil},
	"test_data_type":          {[]byte{packet.Data, 0x00, 0x00, 0x00}, nil},
	"test_kick_type":          {[]byte{packet.Kick, 0x00, 0x00, 0x00}, nil},

	"test_wrong_packet_type": {[]byte{0x06, 0x00, 0x00, 0x00}, packet.ErrWrongPomeloPacketType},
}

var (
	handshakeHeaderPacket = []byte{packet.Handshake, 0x00, 0x00, 0x01, 0x01}
	invalidHeader         = []byte{0xff, 0x00, 0x00, 0x01}
)

var decodeTables = map[string]struct {
	data   []byte
	packet []*packet.Packet
	err    error
}{
	"test_not_enough_bytes": {[]byte{0x01}, nil, nil},
	"test_error_on_forward": {invalidHeader, nil, packet.ErrWrongPomeloPacketType},
	"test_forward":          {handshakeHeaderPacket, []*packet.Packet{{packet.Handshake, 1, []byte{0x01}}}, nil},
	"test_forward_many":     {append(handshakeHeaderPacket, handshakeHeaderPacket...), []*packet.Packet{{packet.Handshake, 1, []byte{0x01}}, {packet.Handshake, 1, []byte{0x01}}}, nil},
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

			sz, typ, err := ppd.forward(bytes.NewBuffer(table.buf))
			if table.err == nil {
				assert.Equal(t, packet.Type(table.buf[0]), typ)
				assert.Equal(t, 0, sz)
			}

			assert.Equal(t, table.err, err)
		})
	}
}

func TestDecode(t *testing.T) {
	t.Parallel()

	for name, table := range decodeTables {
		t.Run(name, func(t *testing.T) {
			ppd := NewPomeloPacketDecoder()

			packet, err := ppd.Decode(table.data)

			assert.Equal(t, table.err, err)
			assert.ElementsMatch(t, table.packet, packet)
		})
	}
}
