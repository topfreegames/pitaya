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

package pitaya

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/session/mocks"
)

func getGroup(n string) *Group {
	return NewGroup(n)
}

func TestNewGroup(t *testing.T) {
	t.Parallel()
	g := getGroup("hello")
	assert.NotNil(t, g)
	assert.Equal(t, "hello", g.name)
}

func TestAdd(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name     string
		frontend bool
		UID      string
		err      error
	}{
		{"frontend_uid", true, "someuid1", nil},
		{"frontend_nouid", true, "", constants.ErrNoUIDBind},
		{"backend_nouid", false, "", constants.ErrNoUIDBind},
		{"backend_uid", false, "ola1", nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g := getGroup("testGroup")
			defer g.Close()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := g.Add(s)
			assert.Equal(t, table.err, err)
			if err == nil {
				assert.True(t, g.Contains(table.UID))
			}
		})
	}
}

func TestContains(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name     string
		frontend bool
		UID      string
		err      error
	}{
		{"frontend_uid", true, "someuid2", nil},
		{"backend_uid", false, "ola2", nil},
		{"backend_nouid", false, "", constants.ErrNoUIDBind},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g := getGroup("testGroup")
			defer g.Close()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			g.Add(s)
			b := g.Contains(s.UID())
			if table.err == nil {
				assert.True(t, b)
			} else {
				assert.False(t, b)
			}
		})
	}
}

func TestLeave(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name     string
		frontend bool
		UID      string
		err      error
	}{
		{"frontend_uid", true, "someuid2", nil},
		{"backend_uid", false, "ola2", nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g := getGroup("testGroup")
			defer g.Close()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			g.Add(s)
			err := g.Leave(s)
			assert.NoError(t, err)
			assert.False(t, g.Contains(table.UID))
		})
	}
}

func TestLeaveAll(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name     string
		frontend bool
		UID      string
		err      error
	}{
		{"frontend_uid", true, "someuid2", nil},
		{"backend_uid", false, "ola2", nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g := getGroup("testGroup")
			defer g.Close()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			g.Add(s)
			err := g.LeaveAll()
			assert.NoError(t, err)
			assert.False(t, g.Contains(table.UID))
		})
	}
}

func TestCount(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name     string
		frontend bool
		UID      string
		err      error
	}{
		{"frontend_uid", true, "someuid2", nil},
		{"backend_uid", false, "ola2", nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g := getGroup("testGroup")
			defer g.Close()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			g.Add(s)
			assert.Equal(t, 1, g.Count())
		})
	}
}

func TestIsClosed(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name     string
		frontend bool
		UID      string
		err      error
	}{
		{"frontend_uid", true, "someuid2", nil},
		{"backend_uid", false, "ola2", nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g := getGroup("testGroup55")
			assert.False(t, g.isClosed())
			g.Close()
			assert.True(t, g.isClosed())
		})
	}
}

func TestMembers(t *testing.T) {
	t.Parallel()
	g := getGroup("testGroup11")
	defer g.Close()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true, "someid1")
	s2 := session.New(mockNetworkEntity, true, "someid2")
	g.Add(s1)
	g.Add(s2)
	assert.ElementsMatch(t, []string{"someid1", "someid2"}, g.Members())
}

func TestBroadcast(t *testing.T) {
	t.Parallel()
	g := getGroup("testGroup22")
	defer g.Close()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true, "someid1")
	s2 := session.New(mockNetworkEntity, true, "someid2")
	g.Add(s1)
	g.Add(s2)
	route := "some.route.bla"
	data := []byte("hellow")
	mockNetworkEntity.EXPECT().Push(route, data).Times(2)
	err := g.Broadcast(route, data)
	assert.NoError(t, err)

	g.Close()
	err = g.Broadcast(route, data)
	assert.EqualError(t, constants.ErrClosedGroup, err.Error())
}

func TestMulticast(t *testing.T) {
	t.Parallel()
	g := getGroup("testGroup22")
	defer g.Close()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true, "someid1")
	s2 := session.New(mockNetworkEntity, true, "someid2")
	g.Add(s1)
	g.Add(s2)
	route := "some.route.bla"
	data := []byte("hellow")
	filter := func(s *session.Session) bool {
		return s.UID() == "someid1"
	}
	mockNetworkEntity.EXPECT().Push(route, data).Times(1)
	err := g.Multicast(route, data, filter)
	assert.NoError(t, err)

	g.Close()
	err = g.Multicast(route, data, filter)
	assert.EqualError(t, constants.ErrClosedGroup, err.Error())
}
