package codec

import (
	"bytes"
	packet2 "github.com/topfreegames/pitaya/v2/pkg/conn/packet"
	"testing"

	"github.com/stretchr/testify/assert"
)

var forwardTables = map[string]struct {
	buf []byte
	err error
}{
	"test_handshake_type":     {[]byte{packet2.Handshake, 0x00, 0x00, 0x00}, nil},
	"test_handshake_ack_type": {[]byte{packet2.HandshakeAck, 0x00, 0x00, 0x00}, nil},
	"test_heartbeat_type":     {[]byte{packet2.Heartbeat, 0x00, 0x00, 0x00}, nil},
	"test_data_type":          {[]byte{packet2.Data, 0x00, 0x00, 0x00}, nil},
	"test_kick_type":          {[]byte{packet2.Kick, 0x00, 0x00, 0x00}, nil},

	"test_wrong_packet_type": {[]byte{0x06, 0x00, 0x00, 0x00}, packet2.ErrWrongPomeloPacketType},
}

var (
	handshakeHeaderPacket = []byte{packet2.Handshake, 0x00, 0x00, 0x01, 0x01}
	invalidHeader         = []byte{0xff, 0x00, 0x00, 0x01}
)

var decodeTables = map[string]struct {
	data   []byte
	packet []*packet2.Packet
	err    error
}{
	"test_not_enough_bytes": {[]byte{0x01}, nil, nil},
	"test_error_on_forward": {invalidHeader, nil, packet2.ErrWrongPomeloPacketType},
	"test_forward":          {handshakeHeaderPacket, []*packet2.Packet{{packet2.Handshake, 1, []byte{0x01}}}, nil},
	"test_forward_many":     {append(handshakeHeaderPacket, handshakeHeaderPacket...), []*packet2.Packet{{packet2.Handshake, 1, []byte{0x01}}, {packet2.Handshake, 1, []byte{0x01}}}, nil},
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
				assert.Equal(t, packet2.Type(table.buf[0]), typ)
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
