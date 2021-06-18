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
	"github.com/topfreegames/pitaya/v2/constants"
	"github.com/topfreegames/pitaya/v2/helpers"
	"github.com/topfreegames/pitaya/v2/networkentity/mocks"
	"github.com/topfreegames/pitaya/v2/protos"
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
		entity      *mocks.MockNetworkEntity
		sessionPool = NewSessionPool().(*sessionPoolImpl)
	)

	tables := map[string]struct {
		sessions func() []Session
		mock     func()
	}{
		"test_close_many_sessions": {
			sessions: func() []Session {
				return []Session{
					sessionPool.NewSession(entity, true, uuid.New().String()),
					sessionPool.NewSession(entity, true, uuid.New().String()),
					sessionPool.NewSession(entity, true, uuid.New().String()),
				}
			},
			mock: func() {
				entity.EXPECT().Close().Times(3)
			},
		},

		"test_close_no_sessions": {
			sessions: func() []Session { return []Session{} },
			mock:     func() {},
		},
	}

	for name, table := range tables {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			entity = mocks.NewMockNetworkEntity(ctrl)
			for _, s := range table.sessions() {
				sessionPool.sessionsByID.Store(s.ID(), s)
				sessionPool.sessionsByUID.Store(s.UID(), s)
			}

			table.mock()

			sessionPool.CloseAll()
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
			sessionPool := NewSessionPool().(*sessionPoolImpl)
			var ss *sessionImpl
			if table.uid != "" {
				ss = sessionPool.NewSession(entity, table.frontend, table.uid).(*sessionImpl)
			} else {
				ss = sessionPool.NewSession(entity, table.frontend).(*sessionImpl)
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
				val, ok := sessionPool.sessionsByID.Load(ss.id)
				assert.True(t, ok)
				assert.Equal(t, val, ss)
			}
		})
	}
}

func TestGetSessionByIDExists(t *testing.T) {
	t.Parallel()

	sessionPool := NewSessionPool()
	expectedSS := sessionPool.NewSession(nil, true)
	ss := sessionPool.GetSessionByID(expectedSS.ID())
	assert.Equal(t, expectedSS, ss)
}

func TestGetSessionByIDDoenstExist(t *testing.T) {
	t.Parallel()
	sessionPool := NewSessionPool()
	ss := sessionPool.GetSessionByID(123456) // huge number to make sure no session with this id
	assert.Nil(t, ss)
}

func TestGetSessionByUIDExists(t *testing.T) {
	uid := uuid.New().String()

	sessionPool := NewSessionPool().(*sessionPoolImpl)
	expectedSS := sessionPool.NewSession(nil, true, uid)
	sessionPool.sessionsByUID.Store(uid, expectedSS)

	ss := sessionPool.GetSessionByUID(uid)
	assert.Equal(t, expectedSS, ss)
}

func TestGetSessionByUIDDoenstExist(t *testing.T) {
	t.Parallel()

	sessionPool := NewSessionPool()
	ss := sessionPool.GetSessionByUID(uuid.New().String())
	assert.Nil(t, ss)
}

func TestKick(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	entity := mocks.NewMockNetworkEntity(ctrl)
	sessionPool := NewSessionPool()
	ss := sessionPool.NewSession(entity, true)
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
			sessionPool := NewSessionPool().(*sessionPoolImpl)
			ss := sessionPool.NewSession(nil, false).(*sessionImpl)
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
	sessionPool := NewSessionPool()
	ss := sessionPool.NewSession(mockEntity, false)
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
	sessionPool := NewSessionPool()
	ss := sessionPool.NewSession(mockEntity, false)
	mid := uint(rand.Int())
	v := someStruct{A: 1, B: "aaa"}
	ctx := context.Background()

	mockEntity.EXPECT().ResponseMID(ctx, mid, v)
	err := ss.ResponseMID(ctx, mid, v)
	assert.NoError(t, err)
}

func TestSessionID(t *testing.T) {
	t.Parallel()

	sessionPool := NewSessionPool().(*sessionPoolImpl)
	ss := sessionPool.NewSession(nil, false).(*sessionImpl)
	ss.id = int64(rand.Uint64())

	id := ss.ID()
	assert.Equal(t, ss.id, id)
}

func TestSessionUID(t *testing.T) {
	t.Parallel()

	sessionPool := NewSessionPool().(*sessionPoolImpl)
	ss := sessionPool.NewSession(nil, false).(*sessionImpl)
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
			sessionPool := NewSessionPool().(*sessionPoolImpl)
			ss := sessionPool.NewSession(nil, false).(*sessionImpl)
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
			sessionPool := NewSessionPool().(*sessionPoolImpl)
			ss := sessionPool.NewSession(nil, false).(*sessionImpl)
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
			sessionPool := NewSessionPool().(*sessionPoolImpl)
			ss := sessionPool.NewSession(nil, false).(*sessionImpl)
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
			sessionPool := NewSessionPool().(*sessionPoolImpl)
			ss := sessionPool.NewSession(nil, false).(*sessionImpl)
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

	sessionPool := NewSessionPool().(*sessionPoolImpl)
	ss := sessionPool.NewSession(nil, false).(*sessionImpl)
	assert.NotNil(t, ss)
	ss.SetFrontendData(frontendID, frontendSessionID)

	assert.Equal(t, frontendID, ss.frontendID)
	assert.Equal(t, frontendSessionID, ss.frontendSessionID)
}

func TestSessionBindFailsWithoutUID(t *testing.T) {
	t.Parallel()

	sessionPool := NewSessionPool()
	ss := sessionPool.NewSession(nil, false)
	assert.NotNil(t, ss)

	err := ss.Bind(nil, "")
	assert.Equal(t, constants.ErrIllegalUID, err)
}

func TestSessionBindFailsIfAlreadyBound(t *testing.T) {
	t.Parallel()

	sessionPool := NewSessionPool().(*sessionPoolImpl)
	ss := sessionPool.NewSession(nil, false).(*sessionImpl)
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
		onSessionBind func(ctx context.Context, s Session) error
		err           error
	}{
		{"successful_on_session_bind", func(ctx context.Context, s Session) error {
			affectedVar = s.UID()
			return nil
		}, nil},
		{"failed_on_session_bind", func(ctx context.Context, s Session) error { return err }, err},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			affectedVar = ""
			sessionPool := NewSessionPool().(*sessionPoolImpl)
			ss := sessionPool.NewSession(nil, true)
			assert.NotNil(t, ss)

			sessionPool.OnSessionBind(table.onSessionBind)
			defer func() { sessionPool.sessionBindCallbacks = make([]func(ctx context.Context, s Session) error, 0) }()

			uid := uuid.New().String()
			err := ss.Bind(nil, uid)

			if table.err != nil {
				assert.Equal(t, table.err, err)
				assert.Empty(t, affectedVar)
				assert.Empty(t, ss.UID())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, uid, affectedVar)
				assert.Equal(t, uid, ss.UID())
			}
		})
	}
}

func TestSessionBindFrontend(t *testing.T) {
	sessionPool := NewSessionPool().(*sessionPoolImpl)
	ss := sessionPool.NewSession(nil, true)
	assert.NotNil(t, ss)

	uid := uuid.New().String()
	err := ss.Bind(nil, uid)
	assert.NoError(t, err)
	assert.Equal(t, uid, ss.UID())

	val, ok := sessionPool.sessionsByUID.Load(uid)
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
			sessionPool := NewSessionPool().(*sessionPoolImpl)
			ss := sessionPool.NewSession(mockEntity, false).(*sessionImpl)
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

			_, ok := sessionPool.sessionsByUID.Load(uid)
			assert.False(t, ok)
		})
	}
}

func TestSessionOnCloseFailsIfBackend(t *testing.T) {
	t.Parallel()

	sessionPool := NewSessionPool()
	ss := sessionPool.NewSession(nil, false)
	assert.NotNil(t, ss)

	err := ss.OnClose(nil)
	assert.Equal(t, constants.ErrOnCloseBackend, err)
}

func TestSessionOnClose(t *testing.T) {
	t.Parallel()

	sessionPool := NewSessionPool()
	ss := sessionPool.NewSession(nil, true)
	assert.NotNil(t, ss)

	expected := false
	f := func() { expected = true }
	err := ss.OnClose(f)
	assert.NoError(t, err)
	assert.Len(t, ss.GetOnCloseCallbacks(), 1)

	ss.GetOnCloseCallbacks()[0]()
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
			sessionPool := NewSessionPool().(*sessionPoolImpl)
			ss := sessionPool.NewSession(mockEntity, true).(*sessionImpl)
			assert.NotNil(t, ss)

			if table.uid != "" {
				sessionPool.sessionsByUID.Store(table.uid, ss)
				ss.uid = table.uid
			}

			mockEntity.EXPECT().Close()
			ss.Close()

			_, ok := sessionPool.sessionsByID.Load(ss.id)
			assert.False(t, ok)

			if table.uid != "" {
				_, ok = sessionPool.sessionsByUID.Load(table.uid)
				assert.False(t, ok)
			}
		})
	}
}

func TestSessionCloseFrontendWithSubscription(t *testing.T) {
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	initialSubs := s.NumSubscriptions()

	conn, err := nats.Connect(fmt.Sprintf("nats://%s", s.Addr()))
	assert.NoError(t, err)
	defer conn.Close()

	subs, err := conn.Subscribe(uuid.New().String(), func(msg *nats.Msg) {})
	assert.NoError(t, err)

	helpers.ShouldEventuallyReturn(t, s.NumSubscriptions, initialSubs+1)
	helpers.ShouldEventuallyReturn(t, conn.NumSubscriptions, 1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEntity := mocks.NewMockNetworkEntity(ctrl)
	sessionPool := NewSessionPool()
	ss := sessionPool.NewSession(mockEntity, true)
	assert.NotNil(t, ss)
	ss.SetSubscriptions([]*nats.Subscription{subs})

	mockEntity.EXPECT().Close()
	ss.Close()

	helpers.ShouldEventuallyReturn(t, s.NumSubscriptions, initialSubs)
	helpers.ShouldEventuallyReturn(t, conn.NumSubscriptions, 0)
}

func TestSessionRemoteAddr(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEntity := mocks.NewMockNetworkEntity(ctrl)
	sessionPool := NewSessionPool()
	ss := sessionPool.NewSession(mockEntity, true)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
	f := func(context.Context, Session) error {
		expected = true
		return nil
	}
	sessionPool := NewSessionPool().(*sessionPoolImpl)
	sessionPool.OnSessionBind(f)
	defer func() { sessionPool.sessionBindCallbacks = make([]func(ctx context.Context, s Session) error, 0) }()
	assert.NotNil(t, sessionPool.OnSessionBind)

	sessionPool.sessionBindCallbacks[0](context.Background(), nil)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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

	sessionPool := NewSessionPool()
	ss := sessionPool.NewSession(nil, true)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(mockEntity, false).(*sessionImpl)
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

	sessionPool := NewSessionPool()
	ss := sessionPool.NewSession(nil, true).(*sessionImpl)
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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, false).(*sessionImpl)

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
			sessionPool := NewSessionPool()
			ss := sessionPool.NewSession(nil, false).(*sessionImpl)
			ss.SetHandshakeData(table.data)
			assert.Equal(t, table.data, ss.handshakeData)
		})
	}
}
