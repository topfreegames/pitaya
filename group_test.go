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
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/groups"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/session/mocks"
)

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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := Add(ctx, "testAdd", s.UID(), table.payload)
			assert.Equal(t, table.err, err)
			if err == nil {
				res, err := Member(ctx, "testAdd", table.UID)
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := SubgroupAdd(ctx, "testSubgroupAdd", "sub", s.UID(), table.payload)
			assert.Equal(t, table.err, err)
			if err == nil {
				res, err := SubgroupMember(ctx, "testSubgroupAdd", "sub", table.UID)
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := Add(ctx, "testContains", s.UID(), nil)
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			b, err := Contains(ctx, "testContains", table.UID)
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := SubgroupAdd(ctx, "testSubgroupContains", "sub", s.UID(), nil)
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			b, err := SubgroupContains(ctx, "testSubgroupContains", "sub", table.UID)
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
			err := Add(ctx, "memberGroups1", table.UID, nil)
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			err = Add(ctx, "memberGroups2", table.UID, nil)
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			res, err := MemberGroups(ctx, table.UID)
			if table.err == nil {
				assert.ElementsMatch(t, []string{"memberGroups1", "memberGroups2"}, res)
			} else {
				assert.Error(t, err)
			}
		})
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			err := SubgroupAdd(ctx, "memberSubgroupTest", "sub1", table.UID, nil)
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			err = SubgroupAdd(ctx, "memberSubgroupTest", "sub2", table.UID, nil)
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			res, err := MemberSubgroups(ctx, "memberSubgroupTest", table.UID)
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := Add(ctx, "testLeave", s.UID(), nil)
			assert.NoError(t, err)
			err = Leave(ctx, "testLeave", s.UID())
			assert.NoError(t, err)
			res, err := Contains(ctx, "testLeave", table.UID)
			assert.NoError(t, err)
			assert.False(t, res)
		})
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := SubgroupAdd(ctx, "testSubgroupLeave", "sub", s.UID(), nil)
			assert.NoError(t, err)
			err = SubgroupLeave(ctx, "testSubgroupLeave", "sub", s.UID())
			assert.NoError(t, err)
			res, err := SubgroupContains(ctx, "testSubgroupLeave", "sub", table.UID)
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := Add(ctx, "testLeaveAll", s.UID(), nil)
			assert.NoError(t, err)
			err = LeaveAll(ctx, "testLeaveAll")
			assert.NoError(t, err)
			res, err := Contains(ctx, "testLeaveAll", table.UID)
			assert.NoError(t, err)
			assert.False(t, res)
		})
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := SubgroupAdd(ctx, "testSubgroupLeaveAll", "sub", s.UID(), nil)
			assert.NoError(t, err)
			err = SubgroupLeaveAll(ctx, "testSubgroupLeaveAll", "sub")
			assert.NoError(t, err)
			res, err := SubgroupContains(ctx, "testSubgroupLeaveAll", "sub", table.UID)
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := Add(ctx, "testCount", s.UID(), nil)
			assert.NoError(t, err)
			res, err := Count(ctx, "testCount")
			assert.NoError(t, err)
			assert.Equal(t, 1, res)
			err = LeaveAll(ctx, "testCount")
			assert.NoError(t, err)
		})
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err := SubgroupAdd(ctx, "testSubgroupCount", "sub", s.UID(), nil)
			assert.NoError(t, err)
			res, err := SubgroupCount(ctx, "testSubgroupCount", "sub")
			assert.NoError(t, err)
			assert.Equal(t, 1, res)
			err = LeaveAll(ctx, "testSubgroupCount")
			assert.NoError(t, err)
		})
	}
}

func TestMember(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s := session.New(mockNetworkEntity, true, "someid1")
	err := Add(ctx, "testMember", s.UID(), nil)
	assert.NoError(t, err)
	res, err := Member(ctx, "testMember", s.UID())
	assert.NoError(t, err)
	assert.Equal(t, &groups.Payload{}, res)
}

func TestSubgroups(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s := session.New(mockNetworkEntity, true, "someid1")
	err := SubgroupAdd(ctx, "testSubgroups", "sub1", s.UID(), nil)
	assert.NoError(t, err)
	err = SubgroupAdd(ctx, "testSubgroups", "sub2", s.UID(), nil)
	assert.NoError(t, err)
	res, err := Subgroups(ctx, "testSubgroups")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"sub1", "sub2"}, res)
}

func TestMembers(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true, "someid1")
	s2 := session.New(mockNetworkEntity, true, "someid2")
	err := Add(ctx, "testMembers", s1.UID(), nil)
	assert.NoError(t, err)
	err = Add(ctx, "testMembers", s2.UID(), nil)
	assert.NoError(t, err)
	res, err := Members(ctx, "testMembers")
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true, "someid1")
	s2 := session.New(mockNetworkEntity, true, "someid2")
	err := SubgroupAdd(ctx, "testSubgroupMembers", "sub", s1.UID(), nil)
	assert.NoError(t, err)
	err = SubgroupAdd(ctx, "testSubgroupMembers", "sub", s2.UID(), nil)
	assert.NoError(t, err)
	res, err := SubgroupMembers(ctx, "testSubgroupMembers", "sub")
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true)
	s2 := session.New(mockNetworkEntity, true)
	err := s1.Bind(ctx, strconv.Itoa(int(s1.ID())))
	assert.NoError(t, err)
	err = s2.Bind(ctx, strconv.Itoa(int(s2.ID())))
	assert.NoError(t, err)
	err = Add(ctx, "testBroadcast", s1.UID(), nil)
	assert.NoError(t, err)
	err = Add(ctx, "testBroadcast", s2.UID(), nil)
	assert.NoError(t, err)
	route := "some.route.bla"
	data := []byte("hellow")
	mockNetworkEntity.EXPECT().Push(route, data).Times(2)
	err = Broadcast(ctx, "testtype", "testBroadcast", route, data)
	assert.NoError(t, err)
}

func TestSubgroupBroadcast(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true)
	s2 := session.New(mockNetworkEntity, true)
	err := s1.Bind(ctx, strconv.Itoa(int(s1.ID())))
	assert.NoError(t, err)
	err = s2.Bind(ctx, strconv.Itoa(int(s2.ID())))
	assert.NoError(t, err)
	err = SubgroupAdd(ctx, "testSubgroupBroadcast", "sub", s1.UID(), nil)
	assert.NoError(t, err)
	err = SubgroupAdd(ctx, "testSubgroupBroadcast", "sub", s2.UID(), nil)
	assert.NoError(t, err)
	route := "some.route.bla"
	data := []byte("hellow")
	mockNetworkEntity.EXPECT().Push(route, data).Times(2)
	err = SubgroupBroadcast(ctx, "testtype", "testSubgroupBroadcast", "sub", route, data)
	assert.NoError(t, err)
}

func TestMulticast(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true)
	s2 := session.New(mockNetworkEntity, true)
	err := s1.Bind(context.Background(), strconv.Itoa(int(s1.ID())))
	assert.NoError(t, err)
	err = s2.Bind(context.Background(), strconv.Itoa(int(s2.ID())))
	assert.NoError(t, err)
	route := "some.route.bla"
	data := []byte("hellow")
	uids := []string{s1.UID(), s2.UID()}
	mockNetworkEntity.EXPECT().Push(route, data).Times(2)
	err = Multicast(ctx, "testtype", route, data, uids)
	assert.NoError(t, err)
}
