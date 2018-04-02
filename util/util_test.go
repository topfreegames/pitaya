package util_test

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/topfreegames/pitaya/helpers"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/mocks"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/util"
)

var update = flag.Bool("update", false, "update .golden files")

type someStruct struct {
	A int
	B string
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	gob.Register(someStruct{})
}

func shutdown() {}

func TestSliceContainsString(t *testing.T) {
	tables := []struct {
		slice []string
		str   string
		ret   bool
	}{
		{[]string{"bla", "ble", "bli"}, "bla", true},
		{[]string{"bl", "ble", "bli"}, "bla", false},
		{[]string{"b", "a", "c"}, "c", true},
		{[]string{"c"}, "c", true},
		{[]string{"c"}, "d", false},
		{[]string{}, "d", false},
		{nil, "d", false},
	}

	for _, table := range tables {
		t.Run(fmt.Sprintf("slice:%s str:%s", table.slice, table.str), func(t *testing.T) {
			res := util.SliceContainsString(table.slice, table.str)
			assert.Equal(t, res, table.ret)
		})
	}
}

func TestSerializeOrRaw(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := mocks.NewMockSerializer(ctrl)
	tables := []struct {
		in  interface{}
		out interface{}
	}{
		{[]byte{1, 2, 3}, []byte{1, 2, 3}},
		{[]byte{3, 2, 3}, []byte{3, 2, 3}},
		{"bla", []byte{1}},
		{"ble", []byte{1}},
	}

	mockSerializer.EXPECT().Marshal("bla").Return([]byte{1}, nil)
	mockSerializer.EXPECT().Marshal("ble").Return([]byte{1}, nil)

	for i, table := range tables {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			res, err := util.SerializeOrRaw(mockSerializer, table.in)
			assert.NoError(t, err)
			assert.Equal(t, table.out, res)
		})
	}
}

func TestGobEncode(t *testing.T) {

	ins := []struct {
		name string
		data []interface{}
	}{
		{"gob_encode_test_1", []interface{}{[]byte{1}, "test", 1}},
		{"gob_encode_test_2", []interface{}{[]byte{1}, someStruct{A: 1, B: "aaa"}, 34}},
		{"gob_encode_test_3", []interface{}{"aaa"}},
	}

	for _, in := range ins {
		t.Run(in.name, func(t *testing.T) {
			b, err := util.GobEncode(in.data...)
			require.NoError(t, err)
			gp := filepath.Join("fixtures", in.name+".golden")
			if *update {
				t.Log("updating golden file")
				helpers.WriteFile(t, gp, b)
			}
			expected := helpers.ReadFile(t, gp)

			assert.Equal(t, expected, b)
		})
	}
}

func TestGobDecode(t *testing.T) {

	ins := []struct {
		name string
		out  []interface{}
	}{
		{"gob_encode_test_1", []interface{}{[]byte{1}, "test", 1}},
		{"gob_encode_test_2", []interface{}{[]byte{1}, someStruct{A: 1, B: "aaa"}, 34}},
		{"gob_encode_test_3", []interface{}{"aaa"}},
	}

	for _, in := range ins {
		t.Run(in.name, func(t *testing.T) {
			gp := filepath.Join("fixtures", in.name+".golden")
			data := helpers.ReadFile(t, gp)
			var reply []interface{}
			err := util.GobDecode(&reply, data)
			require.NoError(t, err)
			assert.Equal(t, reply, in.out)
		})
	}
}

func TestFileExists(t *testing.T) {
	ins := []struct {
		name string
		out  bool
	}{
		{"gob_encode_test_1", true},
		{"gob_encode_test_2", true},
		{"gob_encode_test_3", true},
		{"gob_encode_test_4", false},
	}

	for _, in := range ins {
		t.Run(in.name, func(t *testing.T) {
			gp := filepath.Join("fixtures", in.name+".golden")
			out := util.FileExists(gp)
			assert.Equal(t, out, in.out)
		})
	}

}

func TestGetErrorPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := mocks.NewMockSerializer(ctrl)
	tables := []struct {
		in  error
		out []byte
	}{
		{errors.New("some custom error1"), []byte{0x01}},
		{errors.New("error3"), []byte{0x02}},
		{errors.New("bla"), []byte{0x03}},
		{errors.New(""), []byte{0x04}},
	}
	for i, table := range tables {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			mockSerializer.EXPECT().Marshal(gomock.Any()).Return(table.out, nil)
			b, err := util.GetErrorPayload(mockSerializer, table.in)
			assert.NoError(t, err)
			assert.Equal(t, table.out, b)
		})
	}
}

func TestConvertProtoToMessageType(t *testing.T) {
	tables := []struct {
		in  protos.MsgType
		out message.Type
	}{
		{protos.MsgType_MsgRequest, message.Request},
		{protos.MsgType_MsgNotify, message.Notify},
	}

	for i, table := range tables {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			out := util.ConvertProtoToMessageType(table.in)
			assert.Equal(t, out, table.out)
		})
	}
}
