package message

import (
	"errors"
	"flag"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/helpers"
)

var update = flag.Bool("update", false, "update .golden files")

func resetDicts(t *testing.T) {
	t.Helper()
	routes = make(map[string]uint16)
	codes = make(map[uint16]string)
}

func TestNew(t *testing.T) {
	t.Parallel()

	message := New()

	assert.NotNil(t, message)
}

var encodeTables = map[string]struct {
	message *Message
	routes  map[string]uint16
	msgErr  bool
	gzip    uint8
	err     error
}{
	"test_wrong_type": {&Message{Type: 0xff, Data: []byte{}}, nil, false, 0x0, ErrWrongMessageType},

	"test_request_type": {&Message{Type: Request, Route: "a", Data: []uint8{}}, nil, false, 0x0, nil},
	"test_request_type_compressed": {&Message{Type: Request, Route: "a", Data: []byte{}, compressed: true},
		map[string]uint16{"a": 1}, false, 0x0, nil},

	"test_notify_type": {&Message{Type: Notify, Route: "a", Data: []byte{}}, nil, false, 0x0, nil},
	"test_notify_type_compressed": {&Message{Type: Notify, Route: "a", Data: []byte{}, compressed: true},
		map[string]uint16{"a": 1}, false, 0x0, nil},

	"test_push_type": {&Message{Type: Push, Route: "a", Data: []byte{}}, nil, false, 0x0, nil},
	"test_push_type_compressed": {&Message{Type: Push, Route: "a", Data: []byte{}, compressed: true},
		map[string]uint16{"a": 1}, false, 0x0, nil},

	"test_reponse_type":           {&Message{Type: Response, Data: []byte{}}, nil, false, 0x0, nil},
	"test_reponse_type_with_data": {&Message{Type: Response, Data: []byte{0x01}}, nil, false, 0x0, nil},
	"test_reponse_type_with_id":   {&Message{Type: Response, ID: 129, Data: []byte{}}, nil, false, 0x0, nil},

	"test_reponse_type_with_error": {&Message{Type: Response, Data: []byte{0x01}, Err: true}, nil, true, 0x0, nil},
	"test_must_gzip": {&Message{Type: Response,
		Data: []byte("blablablablablablablablablablablablabla"), Err: true}, nil, true, 0x10, nil},
}

func TestEncode(t *testing.T) {
	for name, table := range encodeTables {
		t.Run(name, func(t *testing.T) {
			message := table.message
			SetDictionary(table.routes)

			messageEncoder := NewMessagesEncoder(table.gzip == 0x10)
			result, err := messageEncoder.Encode(message)
			gp := filepath.Join("fixtures", name+".golden")

			if *update {
				t.Log("updating golden file")
				helpers.WriteFile(t, gp, result)
			}

			expected := helpers.ReadFile(t, gp)

			if err == nil {
				assert.Equal(t, table.gzip, expected[0]&gzipMask)
				assert.Equal(t, expected, result)
			} else {
				assert.Nil(t, result)
			}

			assert.Equal(t, table.err, err)

			resetDicts(t)
		})
	}
}

var decodeTables = map[string]struct {
	message *Message
	routes  map[string]uint16
	msgErr  bool
	gzip    uint8
	err     error
}{
	"test_wrong_type": {&Message{Type: 0xff, Data: []byte{}}, nil, false, 0x0, ErrWrongMessageType},

	"test_request_type": {&Message{Type: Request, Route: "a", Data: []uint8{}}, nil, false, 0x0, nil},
	"test_request_type_compressed": {&Message{Type: Request, Route: "a", Data: []byte{}, compressed: true},
		map[string]uint16{"a": 1}, false, 0x0, nil},

	"test_notify_type": {&Message{Type: Notify, Route: "a", Data: []byte{}}, nil, false, 0x0, nil},
	"test_notify_type_compressed": {&Message{Type: Notify, Route: "a", Data: []byte{}, compressed: true},
		map[string]uint16{"a": 1}, false, 0x0, nil},

	"test_push_type": {&Message{Type: Push, Route: "a", Data: []byte{}}, nil, false, 0x0, nil},
	"test_push_type_compressed": {&Message{Type: Push, Route: "a", Data: []byte{}, compressed: true},
		map[string]uint16{"a": 1}, false, 0x0, nil},

	"test_reponse_type":           {&Message{Type: Response, Data: []byte{}}, nil, false, 0x0, nil},
	"test_reponse_type_with_data": {&Message{Type: Response, Data: []byte{0x01}}, nil, false, 0x0, nil},
	"test_reponse_type_with_id":   {&Message{Type: Response, ID: 129, Data: []byte{}}, nil, false, 0x0, nil},

	"test_reponse_type_with_error": {&Message{Type: Response, Data: []byte{0x01}, Err: true}, nil, true, 0x0, nil},
	"test_must_gzip": {&Message{Type: Response,
		Data: []byte("blablablablablablablablablablablablabla"), Err: true}, nil, true, 0x10, nil},
}

func TestDecode(t *testing.T) {
	for name, table := range decodeTables {
		t.Run(name, func(t *testing.T) {
			SetDictionary(table.routes)

			gp := filepath.Join("fixtures", name+".golden")
			encoded := helpers.ReadFile(t, gp)

			message, err := Decode(encoded)

			if err == nil {
				assert.Equal(t, table.message, message)
			}
			if name == "test_wrong_type" {
				assert.EqualError(t, ErrInvalidMessage, err.Error())
			} else {
				assert.Equal(t, table.err, err)
			}
			resetDicts(t)
		})
	}
}

var dictTables = map[string]struct {
	dicts  []map[string]uint16
	routes map[string]uint16
	codes  map[uint16]string
	err    error
}{
	"test_add_new_route": {[]map[string]uint16{{"a": 1}}, map[string]uint16{"a": 1},
		map[uint16]string{1: "a"}, nil},
	"test_add_new_routes": {[]map[string]uint16{{"a": 1}, {"b": 2}}, map[string]uint16{"a": 1, "b": 2},
		map[uint16]string{1: "a", 2: "b"}, nil},
	"test_override_route": {[]map[string]uint16{{"a": 1}, {"a": 2}}, map[string]uint16{"a": 1},
		map[uint16]string{1: "a"}, errors.New("duplicated route(route: a, code: 1)")},
	"test_override_code": {[]map[string]uint16{{"a": 1}, {"b": 1}}, map[string]uint16{"a": 1},
		map[uint16]string{1: "a"}, errors.New("duplicated route(route: b, code: 1)")},
}

func TestSetDictionaty(t *testing.T) {
	for name, table := range dictTables {
		t.Run(name, func(t *testing.T) {
			for _, dict := range table.dicts {
				SetDictionary(dict)
			}

			assert.Equal(t, table.routes, routes)
			assert.Equal(t, table.codes, codes)

			resetDicts(t)
		})
	}
}
