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
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	nats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/helpers"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/session/mocks"
)

var update = flag.Bool("update", false, "update .golden files")

type someStruct struct {
	A int
	B string
}

type unregisteredStruct struct{}

type mockAddr struct{}

func (ma *mockAddr) Network() string { return "tcp" }
func (ma *mockAddr) String() string  { return "192.0.2.1:25" }

func getEncodedEmptyMap() []byte {
	b, _ := json.Marshal(map[string]interface{}{})
	return b
}

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

func TestCloseAll(t *testing.T) {
	var (
		entity *mocks.MockNetworkEntity
	)

	tables := map[string]struct {
		sessions func() []*Session
		mock     func()
	}{
		"test_close_many_sessions": {
			sessions: func() []*Session {
				return []*Session{
					New(entity, true, uuid.New().String()),
					New(entity, true, uuid.New().String()),
					New(entity, true, uuid.New().String()),
				}
			},
			mock: func() {
				entity.EXPECT().Close().Times(3)
			},
		},

		"test_close_no_sessions": {
			sessions: func() []*Session { return []*Session{} },
			mock:     func() {},
		},
	}

	for name, table := range tables {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			entity = mocks.NewMockNetworkEntity(ctrl)
			for _, s := range table.sessions() {
				sessionsByID.Store(s.ID(), s)
				sessionsByUID.Store(s.UID(), s)
			}

			table.mock()

			CloseAll()
		})
	}
}

func TestNew(t *testing.T) {
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

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
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

func TestKick(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	entity := mocks.NewMockNetworkEntity(ctrl)
	ss := New(entity, true)
	c := context.Background()
	entity.EXPECT().Kick(c)
	entity.EXPECT().Close()
	err := ss.Kick(c)
	assert.NoError(t, err)
}

func TestSessionUpdateEncodedData(t *testing.T) {
	tables := []struct {
		name string
		data map[string]interface{}
	}{
		{"testUpdateEncodedData_1", map[string]interface{}{"byte": []byte{1}}},
		{"testUpdateEncodedData_2", map[string]interface{}{"int": 1}},
		{"testUpdateEncodedData_3", map[string]interface{}{"struct": someStruct{A: 1, B: "aaa"}}},
		{"testUpdateEncodedData_4", map[string]interface{}{"string": "aaa"}},
		{"testUpdateEncodedData_5", map[string]interface{}{}},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, false)
			assert.NotNil(t, ss)

			ss.data = table.data
			err := ss.updateEncodedData()
			assert.NoError(t, err)

			gp := filepath.Join("fixtures", table.name+".golden")
			if *update {
				t.Log("updating golden file")
				helpers.WriteFile(t, gp, ss.encodedData)
			}
			data := helpers.ReadFile(t, gp)

			assert.Equal(t, data, ss.encodedData)
		})
	}
}

func TestSessionPush(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockEntity := mocks.NewMockNetworkEntity(ctrl)
	ss := New(mockEntity, false)
	route := uuid.New().String()
	v := someStruct{A: 1, B: "aaa"}

	mockEntity.EXPECT().Push(route, v)
	err := ss.Push(route, v)
	assert.NoError(t, err)
}

func TestSessionResponseMID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockEntity := mocks.NewMockNetworkEntity(ctrl)
	ss := New(mockEntity, false)
	mid := uint(rand.Int())
	v := someStruct{A: 1, B: "aaa"}
	ctx := context.Background()

	mockEntity.EXPECT().ResponseMID(ctx, mid, v)
	err := ss.ResponseMID(ctx, mid, v)
	assert.NoError(t, err)
}

func TestSessionID(t *testing.T) {
	t.Parallel()

	ss := New(nil, false)
	ss.id = int64(rand.Uint64())

	id := ss.ID()
	assert.Equal(t, ss.id, id)
}

func TestSessionUID(t *testing.T) {
	t.Parallel()

	ss := New(nil, false)
	ss.uid = uuid.New().String()

	uid := ss.UID()
	assert.Equal(t, ss.uid, uid)
}

func TestSessionGetData(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name string
		data map[string]interface{}
	}{
		{"test_1", map[string]interface{}{"byte": []byte{1}, "string": "test", "int": 1}},
		{"test_2", map[string]interface{}{"byte": []byte{1}, "struct": someStruct{A: 1, B: "aaa"}, "int": 34}},
		{"test_3", map[string]interface{}{"string": "aaa"}},
		{"test_4", map[string]interface{}{}},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, false)
			ss.data = table.data

			data := ss.GetData()
			assert.Equal(t, ss.data, data)
		})
	}
}

func TestSessionSetData(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name string
		data map[string]interface{}
	}{
		{"testSessionSetData_1", map[string]interface{}{"byte": []byte{1}}},
		{"testSessionSetData_2", map[string]interface{}{"int": 1}},
		{"testSessionSetData_3", map[string]interface{}{"struct": someStruct{A: 1, B: "aaa"}}},
		{"testSessionSetData_4", map[string]interface{}{"string": "aaa"}},
		{"testSessionSetData_5", map[string]interface{}{}},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, false)
			err := ss.SetData(table.data)
			assert.NoError(t, err)
			assert.Equal(t, table.data, ss.data)

			gp := filepath.Join("fixtures", table.name+".golden")
			if *update {
				t.Log("updating golden file")
				helpers.WriteFile(t, gp, ss.encodedData)
			}
			encodedData := helpers.ReadFile(t, gp)
			assert.Equal(t, encodedData, ss.encodedData)
		})
	}
}

func TestSessionGetEncodedData(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name string
	}{
		{"testUpdateEncodedData_1"},
		{"testUpdateEncodedData_2"},
		{"testUpdateEncodedData_3"},
		{"testUpdateEncodedData_4"},
		{"testUpdateEncodedData_5"},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, false)
			assert.NotNil(t, ss)

			gp := filepath.Join("fixtures", table.name+".golden")
			encodedData := helpers.ReadFile(t, gp)

			ss.encodedData = encodedData
			data := ss.GetDataEncoded()
			assert.Equal(t, ss.encodedData, data)
		})
	}
}

func TestSessionSetEncodedData(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name string
		data map[string]interface{}
	}{
		{"testUpdateEncodedData_2", map[string]interface{}{"int": float64(1)}},
		{"testUpdateEncodedData_3", map[string]interface{}{"struct": map[string]interface{}{"A": float64(1), "B": "aaa"}}},
		{"testUpdateEncodedData_4", map[string]interface{}{"string": "aaa"}},
		{"testUpdateEncodedData_5", map[string]interface{}{}},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, false)
			assert.NotNil(t, ss)

			gp := filepath.Join("fixtures", table.name+".golden")
			encodedData := helpers.ReadFile(t, gp)

			err := ss.SetDataEncoded(encodedData)
			assert.NoError(t, err)
			assert.Equal(t, encodedData, ss.encodedData)
			assert.Equal(t, table.data, ss.data)
		})
	}
}

func TestSessionSetFrontendData(t *testing.T) {
	t.Parallel()

	frontendID := uuid.New().String()
	frontendSessionID := int64(rand.Uint64())

	ss := New(nil, false)
	assert.NotNil(t, ss)
	ss.SetFrontendData(frontendID, frontendSessionID)

	assert.Equal(t, frontendID, ss.frontendID)
	assert.Equal(t, frontendSessionID, ss.frontendSessionID)
}

func TestSessionBindFailsWithoutUID(t *testing.T) {
	t.Parallel()

	ss := New(nil, false)
	assert.NotNil(t, ss)

	err := ss.Bind(nil, "")
	assert.Equal(t, constants.ErrIllegalUID, err)
}

func TestSessionBindFailsIfAlreadyBound(t *testing.T) {
	t.Parallel()

	ss := New(nil, false)
	ss.uid = uuid.New().String()
	assert.NotNil(t, ss)

	err := ss.Bind(nil, uuid.New().String())
	assert.Equal(t, constants.ErrSessionAlreadyBound, err)
}

func TestSessionBindRunsOnSessionBind(t *testing.T) {
	affectedVar := ""
	err := errors.New("some error occured")
	tables := []struct {
		name          string
		onSessionBind func(ctx context.Context, s *Session) error
		err           error
	}{
		{"successful_on_session_bind", func(ctx context.Context, s *Session) error {
			affectedVar = s.uid
			return nil
		}, nil},
		{"failed_on_session_bind", func(ctx context.Context, s *Session) error { return err }, err},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			affectedVar = ""
			ss := New(nil, true)
			assert.NotNil(t, ss)

			OnSessionBind(table.onSessionBind)
			defer func() { sessionBindCallbacks = make([]func(ctx context.Context, s *Session) error, 0) }()

			uid := uuid.New().String()
			err := ss.Bind(nil, uid)

			if table.err != nil {
				assert.Equal(t, table.err, err)
				assert.Empty(t, affectedVar)
				assert.Empty(t, ss.uid)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, uid, affectedVar)
				assert.Equal(t, uid, ss.uid)
			}
		})
	}
}

func TestSessionBindFrontend(t *testing.T) {
	ss := New(nil, true)
	assert.NotNil(t, ss)

	uid := uuid.New().String()
	err := ss.Bind(nil, uid)
	assert.NoError(t, err)
	assert.Equal(t, uid, ss.uid)

	val, ok := sessionsByUID.Load(uid)
	assert.True(t, ok)
	assert.Equal(t, val, ss)
}

func TestSessionBindBackend(t *testing.T) {
	tables := []struct {
		name string
		err  error
	}{
		{"successful_bind_in_front", nil},
		{"failed_bind_in_front", errors.New("failed bind in front")},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockEntity := mocks.NewMockNetworkEntity(ctrl)
			ss := New(mockEntity, false)
			assert.NotNil(t, ss)

			uid := uuid.New().String()
			expectedSessionData := &protos.Session{
				Id:  ss.frontendSessionID,
				Uid: uid,
			}
			ctx := context.Background()
			expectedRequestData, err := proto.Marshal(expectedSessionData)
			assert.NoError(t, err)

			mockEntity.EXPECT().SendRequest(ctx, ss.frontendID, constants.SessionBindRoute, expectedRequestData).Return(&protos.Response{}, table.err)

			err = ss.Bind(ctx, uid)
			assert.Equal(t, table.err, err)

			if table.err == nil {
				assert.Equal(t, uid, ss.uid)
			} else {
				assert.Empty(t, ss.uid)
			}

			_, ok := sessionsByUID.Load(uid)
			assert.False(t, ok)
		})
	}
}

func TestSessionOnCloseFailsIfBackend(t *testing.T) {
	t.Parallel()

	ss := New(nil, false)
	assert.NotNil(t, ss)

	err := ss.OnClose(nil)
	assert.Equal(t, constants.ErrOnCloseBackend, err)
}

func TestSessionOnClose(t *testing.T) {
	t.Parallel()

	ss := New(nil, true)
	assert.NotNil(t, ss)

	expected := false
	f := func() { expected = true }
	err := ss.OnClose(f)
	assert.NoError(t, err)
	assert.Len(t, ss.OnCloseCallbacks, 1)

	ss.OnCloseCallbacks[0]()
	assert.True(t, expected)
}

func TestSessionClose(t *testing.T) {
	tables := []struct {
		name string
		uid  string
	}{
		{"close", ""},
		{"close_bound", uuid.New().String()},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockEntity := mocks.NewMockNetworkEntity(ctrl)
			ss := New(mockEntity, true)
			assert.NotNil(t, ss)

			if table.uid != "" {
				sessionsByUID.Store(table.uid, ss)
				ss.uid = table.uid
			}

			mockEntity.EXPECT().Close()
			ss.Close()

			_, ok := sessionsByID.Load(ss.id)
			assert.False(t, ok)

			if table.uid != "" {
				_, ok = sessionsByUID.Load(table.uid)
				assert.False(t, ok)
			}
		})
	}
}

func TestSessionCloseFrontendWithSubscription(t *testing.T) {
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
        var initialSubs uint32 = s.NumSubscriptions()
	conn, err := nats.Connect(fmt.Sprintf("nats://%s", s.Addr()))
	assert.NoError(t, err)
	defer conn.Close()

	subs, err := conn.Subscribe(uuid.New().String(), func(msg *nats.Msg) {})
	assert.NoError(t, err)
	helpers.ShouldEventuallyReturn(t, s.NumSubscriptions, uint32(initialSubs+1))
	helpers.ShouldEventuallyReturn(t, conn.NumSubscriptions, int(1))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEntity := mocks.NewMockNetworkEntity(ctrl)
	ss := New(mockEntity, true)
	assert.NotNil(t, ss)
	ss.Subscriptions = []*nats.Subscription{subs}

	mockEntity.EXPECT().Close()
	ss.Close()

	helpers.ShouldEventuallyReturn(t, s.NumSubscriptions, uint32(initialSubs))
	helpers.ShouldEventuallyReturn(t, conn.NumSubscriptions, int(0))
}

func TestSessionRemoteAddr(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEntity := mocks.NewMockNetworkEntity(ctrl)
	ss := New(mockEntity, true)
	assert.NotNil(t, ss)

	expectedAddr := &mockAddr{}
	mockEntity.EXPECT().RemoteAddr().Return(expectedAddr)
	addr := ss.RemoteAddr()
	assert.Equal(t, expectedAddr, addr)
}

func TestSessionSet(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name   string
		val    interface{}
		errStr string
	}{
		{"ok", "val", ""},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)
			err := ss.Set("key", table.val)
			if table.errStr == "" {
				assert.NoError(t, err)
				val, ok := ss.data["key"]
				assert.True(t, ok)
				assert.Equal(t, table.val, val)
				assert.NotEmpty(t, ss.encodedData)
			}
		})
	}
}

func TestSessionRemove(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name string
		val  interface{}
	}{
		{"existent", "val"},
		{"unexistent", nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
				assert.NotEmpty(t, ss.encodedData)
			}

			err := ss.Remove("key")
			assert.NoError(t, err)
			assert.Empty(t, ss.data)

			expectedEncoded := getEncodedEmptyMap()
			assert.Equal(t, expectedEncoded, ss.encodedData)
		})
	}
}

func TestOnSessionBind(t *testing.T) {
	expected := false
	f := func(context.Context, *Session) error {
		expected = true
		return nil
	}
	OnSessionBind(f)
	defer func() { sessionBindCallbacks = make([]func(ctx context.Context, s *Session) error, 0) }()
	assert.NotNil(t, OnSessionBind)

	sessionBindCallbacks[0](context.Background(), nil)
	assert.True(t, expected)
}

func TestSessionHasKey(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name string
		val  interface{}
	}{
		{"existent", "val"},
		{"unexistent", nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			exists := ss.HasKey("key")
			assert.Equal(t, table.val != nil, exists)
		})
	}
}

func TestSessionGet(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name string
		val  interface{}
	}{
		{"existent", "val"},
		{"unexistent", nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Get("key")
			assert.Equal(t, table.val, val)
		})
	}
}

func TestSessionInt(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name         string
		val          interface{}
		shouldReturn bool
	}{
		{"existent", 1, true},
		{"existent_wrong_type", "val", false},
		{"unexistent", nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Int("key")
			if table.shouldReturn {
				assert.Equal(t, table.val, val)
			} else {
				assert.Equal(t, 0, val)
			}
		})
	}
}

func TestSessionInt8(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name         string
		val          interface{}
		shouldReturn bool
	}{
		{"existent", int8(1), true},
		{"existent_wrong_type", "val", false},
		{"unexistent", nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Int8("key")
			if table.shouldReturn {
				assert.Equal(t, table.val, val)
			} else {
				assert.Equal(t, int8(0), val)
			}
		})
	}
}

func TestSessionInt16(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name         string
		val          interface{}
		shouldReturn bool
	}{
		{"existent", int16(1), true},
		{"existent_wrong_type", "val", false},
		{"unexistent", nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Int16("key")
			if table.shouldReturn {
				assert.Equal(t, table.val, val)
			} else {
				assert.Equal(t, int16(0), val)
			}
		})
	}
}

func TestSessionInt32(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name         string
		val          interface{}
		shouldReturn bool
	}{
		{"existent", int32(1), true},
		{"existent_wrong_type", "val", false},
		{"unexistent", nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Int32("key")
			if table.shouldReturn {
				assert.Equal(t, table.val, val)
			} else {
				assert.Equal(t, int32(0), val)
			}
		})
	}
}

func TestSessionInt64(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name         string
		val          interface{}
		shouldReturn bool
	}{
		{"existent", int64(1), true},
		{"existent_wrong_type", "val", false},
		{"unexistent", nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Int64("key")
			if table.shouldReturn {
				assert.Equal(t, table.val, val)
			} else {
				assert.Equal(t, int64(0), val)
			}
		})
	}
}

func TestSessionUint(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name         string
		val          interface{}
		shouldReturn bool
	}{
		{"existent", uint(1), true},
		{"existent_wrong_type", "val", false},
		{"unexistent", nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Uint("key")
			if table.shouldReturn {
				assert.Equal(t, table.val, val)
			} else {
				assert.Equal(t, uint(0), val)
			}
		})
	}
}

func TestSessionUint8(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name         string
		val          interface{}
		shouldReturn bool
	}{
		{"existent", uint8(1), true},
		{"existent_wrong_type", "val", false},
		{"unexistent", nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Uint8("key")
			if table.shouldReturn {
				assert.Equal(t, table.val, val)
			} else {
				assert.Equal(t, uint8(0), val)
			}
		})
	}
}

func TestSessionUint16(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name         string
		val          interface{}
		shouldReturn bool
	}{
		{"existent", uint16(1), true},
		{"existent_wrong_type", "val", false},
		{"unexistent", nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Uint16("key")
			if table.shouldReturn {
				assert.Equal(t, table.val, val)
			} else {
				assert.Equal(t, uint16(0), val)
			}
		})
	}
}

func TestSessionUint32(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name         string
		val          interface{}
		shouldReturn bool
	}{
		{"existent", uint32(1), true},
		{"existent_wrong_type", "val", false},
		{"unexistent", nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Uint32("key")
			if table.shouldReturn {
				assert.Equal(t, table.val, val)
			} else {
				assert.Equal(t, uint32(0), val)
			}
		})
	}
}

func TestSessionUint64(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name         string
		val          interface{}
		shouldReturn bool
	}{
		{"existent", uint64(1), true},
		{"existent_wrong_type", "val", false},
		{"unexistent", nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Uint64("key")
			if table.shouldReturn {
				assert.Equal(t, table.val, val)
			} else {
				assert.Equal(t, uint64(0), val)
			}
		})
	}
}

func TestSessionFloat32(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name         string
		val          interface{}
		shouldReturn bool
	}{
		{"existent", float32(1), true},
		{"existent_wrong_type", "val", false},
		{"unexistent", nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Float32("key")
			if table.shouldReturn {
				assert.Equal(t, table.val, val)
			} else {
				assert.Equal(t, float32(0), val)
			}
		})
	}
}

func TestSessionFloat64(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name         string
		val          interface{}
		shouldReturn bool
	}{
		{"existent", float64(1), true},
		{"existent_wrong_type", "val", false},
		{"unexistent", nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Float64("key")
			if table.shouldReturn {
				assert.Equal(t, table.val, val)
			} else {
				assert.Equal(t, float64(0), val)
			}
		})
	}
}

func TestSessionString(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name         string
		val          interface{}
		shouldReturn bool
	}{
		{"existent", "val", true},
		{"existent_wrong_type", 1, false},
		{"unexistent", nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.String("key")
			if table.shouldReturn {
				assert.Equal(t, table.val, val)
			} else {
				assert.Equal(t, "", val)
			}
		})
	}
}

func TestSessionValue(t *testing.T) {
	t.Parallel()

	tables := []struct {
		name string
		val  interface{}
	}{
		{"existent", "val"},
		{"unexistent", nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, true)
			assert.NotNil(t, ss)

			if table.val != nil {
				err := ss.Set("key", table.val)
				assert.NoError(t, err)
				assert.NotEmpty(t, ss.data)
			}

			val := ss.Value("key")
			assert.Equal(t, table.val, val)
		})
	}
}

func TestSessionPushToFrontFailsIfFrontend(t *testing.T) {
	t.Parallel()

	ss := New(nil, true)
	assert.NotNil(t, ss)

	err := ss.PushToFront(nil)
	assert.Equal(t, constants.ErrFrontSessionCantPushToFront, err)
}

func TestSessionPushToFront(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name string
		err  error
	}{
		{"successful_request", nil},
		{"failed_request", errors.New("failed bind in front")},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			mockEntity := mocks.NewMockNetworkEntity(ctrl)
			ss := New(mockEntity, false)
			assert.NotNil(t, ss)
			ss.Set("key", "val")
			uid := uuid.New().String()
			ss.uid = uid

			expectedSessionData := &protos.Session{
				Id:   ss.frontendSessionID,
				Uid:  uid,
				Data: ss.encodedData,
			}
			expectedRequestData, err := proto.Marshal(expectedSessionData)
			assert.NoError(t, err)
			ctx := context.Background()
			mockEntity.EXPECT().SendRequest(ctx, ss.frontendID, constants.SessionPushRoute, expectedRequestData).Return(nil, table.err)

			err = ss.PushToFront(ctx)
			assert.Equal(t, table.err, err)
		})
	}
}

func TestSessionClear(t *testing.T) {
	t.Parallel()

	ss := New(nil, true)
	assert.NotNil(t, ss)

	ss.uid = uuid.New().String()
	err := ss.Set("key", "val")
	assert.NoError(t, err)
	assert.NotEmpty(t, ss.data)
	assert.NotEmpty(t, ss.encodedData)

	ss.Clear()
	assert.Empty(t, ss.data)

	expectedEncoded := getEncodedEmptyMap()
	assert.Equal(t, expectedEncoded, ss.encodedData)
}

func TestSessionGetHandshakeData(t *testing.T) {
	t.Parallel()

	data1 := &HandshakeData{
		Sys: HandshakeClientData{
			Platform:    "macos",
			LibVersion:  "2.3.2",
			BuildNumber: "20",
			Version:     "14.0.2",
		},
		User: make(map[string]interface{}),
	}
	data2 := &HandshakeData{
		Sys: HandshakeClientData{
			Platform:    "windows",
			LibVersion:  "2.3.10",
			BuildNumber: "",
			Version:     "ahaha",
		},
		User: map[string]interface{}{
			"ababa": make(map[string]interface{}),
			"pepe":  1,
		},
	}
	tables := []struct {
		name string
		data *HandshakeData
	}{
		{"test_1", data1},
		{"test_2", data2},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, false)

			assert.Nil(t, ss.GetHandshakeData())

			ss.handshakeData = table.data

			assert.Equal(t, ss.GetHandshakeData(), table.data)
		})
	}
}

func TestSessionSetHandshakeData(t *testing.T) {
	t.Parallel()

	data1 := &HandshakeData{
		Sys: HandshakeClientData{
			Platform:    "macos",
			LibVersion:  "2.3.2",
			BuildNumber: "20",
			Version:     "14.0.2",
		},
		User: make(map[string]interface{}),
	}
	data2 := &HandshakeData{
		Sys: HandshakeClientData{
			Platform:    "windows",
			LibVersion:  "2.3.10",
			BuildNumber: "",
			Version:     "ahaha",
		},
		User: map[string]interface{}{
			"ababa": make(map[string]interface{}),
			"pepe":  1,
		},
	}
	tables := []struct {
		name string
		data *HandshakeData
	}{
		{"testSessionSetData_1", data1},
		{"testSessionSetData_2", data2},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ss := New(nil, false)
			ss.SetHandshakeData(table.data)
			assert.Equal(t, table.data, ss.handshakeData)
		})
	}
}
