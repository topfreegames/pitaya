package message

import (
	"errors"
	"flag"
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
	err     error
}{
	"test_wrong_type": {&Message{Type: 0xff}, nil, ErrWrongMessageType},

	"test_request_type": {&Message{Type: Request, Route: "a"}, nil, nil},
	"test_request_type_compressed": {&Message{Type: Request, Route: "a", Data: []byte{}, compressed: true},
		map[string]uint16{"a": 1}, nil},

	"test_notify_type": {&Message{Type: Notify, Route: "a", Data: []byte{}}, nil, nil},
	"test_notify_type_compressed": {&Message{Type: Notify, Route: "a", Data: []byte{}, compressed: true},
		map[string]uint16{"a": 1}, nil},

	"test_push_type": {&Message{Type: Push, Route: "a", Data: []byte{}}, nil, nil},
	"test_push_type_compressed": {&Message{Type: Push, Route: "a", Data: []byte{}, compressed: true},
		map[string]uint16{"a": 1}, nil},

	"test_reponse_type":           {&Message{Type: Response, Data: []byte{}}, nil, nil},
	"test_reponse_type_with_data": {&Message{Type: Response, Data: []byte{0x01}}, nil, nil},
	"test_reponse_type_with_id":   {&Message{Type: Response, ID: 129, Data: []byte{}}, nil, nil},
}

func TestEncode(t *testing.T) {
	for name, table := range encodeTables {
		t.Run(name, func(t *testing.T) {
			message := table.message
			SetDictionary(table.routes)

			result, err := message.Encode()
			gp := helpers.FixtureGoldenFileName(t, t.Name())

			if *update {
				t.Log("updating golden file")
				helpers.WriteFile(t, gp, result)
			}

			expected := helpers.ReadFile(t, gp)

			if err == nil {
				assert.Equal(t, expected, result)
			} else {
				assert.Nil(t, result)
			}

			assert.Equal(t, table.err, err)

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

var decodeTables = map[string]struct {
	encodedMessage *Message
	decodedMessage *Message
	routes         map[string]uint16
	err            error
}{
	"test_request_type": {&Message{Type: Request, Data: []byte{}},
		&Message{Type: Request, Data: []byte{}}, nil, nil},
	"test_request_type_compressed": {&Message{Type: Request, Route: "a", Data: []byte{}},
		&Message{Type: Request, Route: "a", Data: []byte{}, compressed: true}, map[string]uint16{"a": 1}, nil},

	"test_notify_type": {&Message{Type: Notify, Data: []byte{}},
		&Message{Type: Notify, Data: []byte{}}, nil, nil},
	"test_notify_type_compressed": {&Message{Type: Notify, Route: "a", Data: []byte{}},
		&Message{Type: Notify, Route: "a", Data: []byte{}, compressed: true}, map[string]uint16{"a": 1}, nil},

	"test_push_type": {&Message{Type: Push, Data: []byte{}},
		&Message{Type: Push, Data: []byte{}}, nil, nil},
	"test_push_type_compressed": {&Message{Type: Push, Route: "a", Data: []byte{}},
		&Message{Type: Push, Route: "a", Data: []byte{}, compressed: true}, map[string]uint16{"a": 1}, nil},
	"test_push_type_compressed_code_2": {&Message{Type: Push, Route: "a", Data: []byte{}},
		nil, map[string]uint16{"a": 2}, ErrRouteInfoNotFound},

	"test_reponse_type": {&Message{Type: Response, Data: []byte{}},
		&Message{Type: Response, Data: []byte{}}, nil, nil},
	"test_reponse_type_with_data": {&Message{Type: Response, Data: []byte{0x01}},
		&Message{Type: Response, Data: []byte{0x01}}, nil, nil},
	"test_reponse_type_with_id": {&Message{Type: Response, ID: 129, Data: []byte{}},
		&Message{Type: Response, ID: 129, Data: []byte{}}, nil, nil},
}

func TestDecode(t *testing.T) {
	for name, table := range decodeTables {
		t.Run(name, func(t *testing.T) {

			if *update {
				SetDictionary(map[string]uint16{"a": 1})

				result, err := table.encodedMessage.Encode()
				assert.NoError(t, err)

				gp := helpers.FixtureGoldenFileName(t, t.Name())

				t.Log("updating golden file", gp)
				helpers.WriteFile(t, gp, result)
				resetDicts(t)
			}

			SetDictionary(table.routes)

			gp := helpers.FixtureGoldenFileName(t, t.Name())
			encoded := helpers.ReadFile(t, gp)

			message, err := Decode(encoded)

			assert.Equal(t, table.decodedMessage, message)
			assert.Equal(t, table.err, err)

			resetDicts(t)
		})
	}
}
