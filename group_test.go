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
	"context"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/groups"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/session/mocks"
)

func getGroup(n string, t *testing.T) *Group {
	gs, err := groups.NewEtcdGroupService(config.NewConfig(), nil)
	assert.NoError(t, err)
	return NewGroup(n, gs)
}

func TestNewGroup(t *testing.T) {
	t.Parallel()
	g := getGroup("testNewGroup", t)
	assert.NotNil(t, g)
	assert.Equal(t, "testNewGroup", g.name)
}

func TestAdd(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	tables := []struct {
		name     string
		frontend bool
		UID      string
		payload  *groups.Payload
		err      error
	}{
		{"frontend_uid", true, "someuid1", &groups.Payload{Metadata: "leader"}, nil},
		{"frontend_nouid", true, "", nil, constants.ErrNoUIDBind},
		{"backend_nouid", false, "", nil, constants.ErrNoUIDBind},
		{"backend_uid", false, "ola1", &groups.Payload{Metadata: "betatester"}, nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g := getGroup("testAdd", t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := g.Add(ctx, s.UID(), table.payload)
			assert.Equal(t, table.err, err)
			if err == nil {
				res, err := g.Member(ctx, table.UID)
				if table.err == nil {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
				}
				assert.Equal(t, table.payload, res)
			}
		})
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g := getGroup("testSubgroupAdd", t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := g.SubgroupAdd(ctx, "sub", s.UID(), table.payload)
			assert.Equal(t, table.err, err)
			if err == nil {
				res, err := g.SubgroupMember(ctx, "sub", table.UID)
				if table.err == nil {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
				}
				assert.Equal(t, table.payload, res)
			}
		})
	}
}

func TestContains(t *testing.T) {
	ctx := context.Background()
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
			g := getGroup("testContains", t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := g.Add(ctx, s.UID(), nil)
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			b, err := g.Contains(ctx, table.UID)
			if table.err == nil {
				assert.True(t, b)
				assert.NoError(t, err)
			} else {
				assert.False(t, b)
				assert.Error(t, err)
			}
		})
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g := getGroup("testSubgroupContains", t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := g.SubgroupAdd(ctx, "sub", s.UID(), nil)
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			b, err := g.SubgroupContains(ctx, "sub", table.UID)
			if table.err == nil {
				assert.True(t, b)
				assert.NoError(t, err)
			} else {
				assert.False(t, b)
				assert.Error(t, err)
			}
		})
	}
}

func TestMemberGroups(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	tables := []struct {
		name string
		UID  string
		err  error
	}{
		{"frontend_uid", "uniqueMember1", nil},
		{"backend_uid", "uniqueMember2", nil},
		{"backend_nouid", "", constants.ErrNoUIDBind},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g1 := getGroup("memberGroups1", t)
			g2 := getGroup("memberGroups2", t)
			err := g1.Add(ctx, table.UID, nil)
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			err = g2.Add(ctx, table.UID, nil)
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			res, err := g1.MemberGroups(ctx, table.UID)
			if table.err == nil {
				assert.ElementsMatch(t, []string{"memberGroups1", "memberGroups2"}, res)
			} else {
				assert.Error(t, err)
			}
		})
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g := getGroup("memberSubgroupTest", t)
			err := g.SubgroupAdd(ctx, "sub1", table.UID, nil)
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			err = g.SubgroupAdd(ctx, "sub2", table.UID, nil)
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			res, err := g.MemberSubgroups(ctx, table.UID)
			if table.err == nil {
				assert.ElementsMatch(t, []string{"sub1", "sub2"}, res)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestLeave(t *testing.T) {
	ctx := context.Background()
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
			g := getGroup("testLeave", t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := g.Add(ctx, s.UID(), nil)
			assert.NoError(t, err)
			err = g.Leave(ctx, s.UID())
			assert.NoError(t, err)
			res, err := g.Contains(ctx, table.UID)
			assert.NoError(t, err)
			assert.False(t, res)
		})
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g := getGroup("testSubgroupLeave", t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := g.SubgroupAdd(ctx, "sub", s.UID(), nil)
			assert.NoError(t, err)
			err = g.SubgroupLeave(ctx, "sub", s.UID())
			assert.NoError(t, err)
			res, err := g.SubgroupContains(ctx, "sub", table.UID)
			assert.NoError(t, err)
			assert.False(t, res)
		})
	}
}

func TestLeaveAll(t *testing.T) {
	ctx := context.Background()
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
			g := getGroup("testLeaveAll", t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := g.Add(ctx, s.UID(), nil)
			assert.NoError(t, err)
			err = g.LeaveAll(ctx)
			assert.NoError(t, err)
			res, err := g.Contains(ctx, table.UID)
			assert.NoError(t, err)
			assert.False(t, res)
		})
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g := getGroup("testSubgroupLeaveAll", t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := g.SubgroupAdd(ctx, "sub", s.UID(), nil)
			assert.NoError(t, err)
			err = g.SubgroupLeaveAll(ctx, "sub")
			assert.NoError(t, err)
			res, err := g.SubgroupContains(ctx, "sub", table.UID)
			assert.NoError(t, err)
			assert.False(t, res)
		})
	}
}

func TestCount(t *testing.T) {
	ctx := context.Background()
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
			g := getGroup("testCount", t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := g.Add(ctx, s.UID(), nil)
			assert.NoError(t, err)
			res, err := g.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 1, res)
			err = g.LeaveAll(ctx)
			assert.NoError(t, err)
		})
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			g := getGroup("testSubgroupCount", t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := g.SubgroupAdd(ctx, "sub", s.UID(), nil)
			assert.NoError(t, err)
			res, err := g.SubgroupCount(ctx, "sub")
			assert.NoError(t, err)
			assert.Equal(t, 1, res)
			err = g.LeaveAll(ctx)
			assert.NoError(t, err)
		})
	}
}

func TestMember(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	g := getGroup("testMember", t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s := session.New(mockNetworkEntity, true, "someid1")
	err := g.Add(ctx, s.UID(), nil)
	assert.NoError(t, err)
	res, err := g.Member(ctx, s.UID())
	assert.NoError(t, err)
	assert.Equal(t, &groups.Payload{}, res)
}

func TestSubgroups(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	g := getGroup("testSubgroups", t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s := session.New(mockNetworkEntity, true, "someid1")
	err := g.SubgroupAdd(ctx, "sub1", s.UID(), nil)
	assert.NoError(t, err)
	err = g.SubgroupAdd(ctx, "sub2", s.UID(), nil)
	assert.NoError(t, err)
	res, err := g.Subgroups(ctx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"sub1", "sub2"}, res)
}

func TestMembers(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	g := getGroup("testMembers", t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true, "someid1")
	s2 := session.New(mockNetworkEntity, true, "someid2")
	err := g.Add(ctx, s1.UID(), nil)
	assert.NoError(t, err)
	err = g.Add(ctx, s2.UID(), nil)
	assert.NoError(t, err)
	res, err := g.Members(ctx)
	assert.NoError(t, err)
	uids := make([]string, 0, len(res))
	for uid := range res {
		uids = append(uids, uid)
	}
	assert.ElementsMatch(t, []string{"someid1", "someid2"}, uids)
}

func TestSubgroupMembers(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	g := getGroup("testSubgroupMembers", t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true, "someid1")
	s2 := session.New(mockNetworkEntity, true, "someid2")
	err := g.SubgroupAdd(ctx, "sub", s1.UID(), nil)
	assert.NoError(t, err)
	err = g.SubgroupAdd(ctx, "sub", s2.UID(), nil)
	assert.NoError(t, err)
	res, err := g.SubgroupMembers(ctx, "sub")
	assert.NoError(t, err)
	uids := make([]string, 0, len(res))
	for uid := range res {
		uids = append(uids, uid)
	}
	assert.ElementsMatch(t, []string{"someid1", "someid2"}, uids)
}

func TestBroadcast(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	g := getGroup("testBroadcast", t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true)
	s2 := session.New(mockNetworkEntity, true)
	err := s1.Bind(ctx, strconv.Itoa(int(s1.ID())))
	assert.NoError(t, err)
	err = s2.Bind(ctx, strconv.Itoa(int(s2.ID())))
	assert.NoError(t, err)
	err = g.Add(ctx, s1.UID(), nil)
	assert.NoError(t, err)
	err = g.Add(ctx, s2.UID(), nil)
	assert.NoError(t, err)
	route := "some.route.bla"
	data := []byte("hellow")
	mockNetworkEntity.EXPECT().Push(route, data).Times(2)
	err = g.Broadcast(ctx, "testtype", route, data)
	assert.NoError(t, err)
}

func TestSubgroupBroadcast(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	g := getGroup("testSubgroupBroadcast", t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true)
	s2 := session.New(mockNetworkEntity, true)
	err := s1.Bind(ctx, strconv.Itoa(int(s1.ID())))
	assert.NoError(t, err)
	err = s2.Bind(ctx, strconv.Itoa(int(s2.ID())))
	assert.NoError(t, err)
	err = g.SubgroupAdd(ctx, "sub", s1.UID(), nil)
	assert.NoError(t, err)
	err = g.SubgroupAdd(ctx, "sub", s2.UID(), nil)
	assert.NoError(t, err)
	route := "some.route.bla"
	data := []byte("hellow")
	mockNetworkEntity.EXPECT().Push(route, data).Times(2)
	err = g.SubgroupBroadcast(ctx, "testtype", "sub", route, data)
	assert.NoError(t, err)
}

func TestMulticast(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	g := getGroup("testMulticast", t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true)
	s2 := session.New(mockNetworkEntity, true)
	err := s1.Bind(context.Background(), strconv.Itoa(int(s1.ID())))
	assert.NoError(t, err)
	err = s2.Bind(context.Background(), strconv.Itoa(int(s2.ID())))
	assert.NoError(t, err)
	err = g.Add(ctx, s1.UID(), nil)
	assert.NoError(t, err)
	err = g.Add(ctx, s2.UID(), nil)
	assert.NoError(t, err)
	route := "some.route.bla"
	data := []byte("hellow")
	uids := []string{s1.UID(), s2.UID()}
	mockNetworkEntity.EXPECT().Push(route, data).Times(2)
	err = g.Multicast(ctx, "testtype", route, data, uids)
	assert.NoError(t, err)
}
