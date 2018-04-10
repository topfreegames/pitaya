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

package session

import (
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/helpers"
	"github.com/topfreegames/pitaya/session/mocks"
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

func TestNewSessionIDService(t *testing.T) {
	t.Parallel()

	sessionIDService := newSessionIDService()
	assert.NotNil(t, sessionIDService)
	assert.EqualValues(t, 0, sessionIDService.sid)
}

func TestSessionIDServiceSessionID(t *testing.T) {
	t.Parallel()

	sessionIDService := newSessionIDService()
	sessionID := sessionIDService.sessionID()
	assert.EqualValues(t, 1, sessionID)
}

func TestNew(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name     string
		frontend bool
		uid      string
	}{
		{"test_frontend", true, ""},
		{"test_backend", false, ""},
		{"test_frontend_with_uid", true, uuid.New().String()},
		{"test_backend_with_uid", false, uuid.New().String()},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			entity := mocks.NewMockNetworkEntity(ctrl)
			var ss *Session
			if table.uid != "" {
				ss = New(entity, table.frontend, table.uid)
			} else {
				ss = New(entity, table.frontend)
			}
			assert.NotZero(t, ss.id)
			assert.Equal(t, entity, ss.entity)
			assert.Empty(t, ss.data)
			assert.InDelta(t, time.Now().Unix(), ss.lastTime, 1)
			assert.Empty(t, ss.OnCloseCallbacks)
			assert.Equal(t, table.frontend, ss.IsFrontend)

			if len(table.uid) > 0 {
				assert.Equal(t, table.uid[0], ss.uid[0])
			}

			if table.frontend {
				val, ok := sessionsByID.Load(ss.id)
				assert.True(t, ok)
				assert.Equal(t, val, ss)
			}
		})
	}
}

func TestGetSessionByIDExists(t *testing.T) {
	t.Parallel()

	expectedSS := New(nil, true)
	ss := GetSessionByID(expectedSS.id)
	assert.Equal(t, expectedSS, ss)
}

func TestGetSessionByIDDoenstExist(t *testing.T) {
	t.Parallel()

	ss := GetSessionByID(123456) // huge number to make sure no session with this id
	assert.Nil(t, ss)
}

func TestGetSessionByUIDExists(t *testing.T) {
	t.Parallel()

	uid := uuid.New().String()
	expectedSS := New(nil, true, uid)
	sessionsByUID.Store(uid, expectedSS)

	ss := GetSessionByUID(uid)
	assert.Equal(t, expectedSS, ss)
}

func TestGetSessionByUIDDoenstExist(t *testing.T) {
	t.Parallel()

	ss := GetSessionByUID(uuid.New().String())
	assert.Nil(t, ss)
}

func TestUpdateEncondedData(t *testing.T) {
	tables := []struct {
		name string
		data map[string]interface{}
	}{
		{"testUpdateEncondedData_1", map[string]interface{}{"byte": []byte{1}, "string": "test", "int": 1}},
		{"testUpdateEncondedData_2", map[string]interface{}{"byte": []byte{1}, "struct": someStruct{A: 1, B: "aaa"}, "int": 34}},
		{"testUpdateEncondedData_3", map[string]interface{}{"string": "aaa"}},
		{"testUpdateEncondedData_4", map[string]interface{}{}},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, false)
			assert.NotNil(t, ss)

			ss.data = table.data
			err := ss.updateEncodedData()
			assert.NoError(t, err)

			gp := filepath.Join("fixtures", table.name+".golden")
			fmt.Println(gp)
			if *update {
				t.Log("updating golden file")
				helpers.WriteFile(t, gp, ss.encodedData)
			}
			data := helpers.ReadFile(t, gp)

			assert.Equal(t, data, ss.encodedData)
		})
	}
}
