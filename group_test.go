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
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/session/mocks"
)

func TestCreateDuplicatedGroup(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	err := GroupCreate(ctx, "testCreateDuplicatedGroup")
	assert.NoError(t, err)
	err = GroupCreate(ctx, "testCreateDuplicatedGroup")
	assert.Error(t, err)
	assert.Equal(t, constants.ErrGroupAlreadyExists, err)
}

func TestCreateGroup(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	err := GroupCreate(ctx, "testCreateGroup")
	assert.NoError(t, err)
	count, err := GroupCountMembers(ctx, "testCreateGroup")
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	err = GroupRenewTTL(ctx, "testCreateGroup")
	assert.Error(t, err)
}

func TestCreateGroupWithTTL(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	err := GroupCreateWithTTL(ctx, "testCreateGroupWithTTL", 10)
	assert.NoError(t, err)
	count, err := GroupCountMembers(ctx, "testCreateGroupWithTTL")
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	err = GroupRenewTTL(ctx, "testCreateGroupWithTTL")
	assert.NoError(t, err)
}

func TestGroupAddMember(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	tables := []struct {
		name     string
		frontend bool
		UID      string
		err      error
	}{
		{"frontend_uid", true, "someuid1", nil},
		{"frontend_nouid", true, "", constants.ErrEmptyUID},
		{"backend_nouid", false, "", constants.ErrEmptyUID},
		{"backend_uid", false, "ola1", nil},
	}

	err := GroupCreate(ctx, "testGroupAddMember")
	assert.NoError(t, err)

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err = GroupAddMember(ctx, "testGroupAddMember", s.UID())
			assert.Equal(t, table.err, err)
			if err == nil {
				_, err := GroupContainsMember(ctx, "testGroupAddMember", table.UID)
				if table.err == nil {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
				}
			}
		})
	}
}

func TestGroupAddDuplicatedMember(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	err := GroupCreate(ctx, "testGroupAddDuplicatedMember")
	assert.NoError(t, err)
	err = GroupAddMember(ctx, "testGroupAddDuplicatedMember", "duplicatedUid")
	assert.NoError(t, err)
	err = GroupAddMember(ctx, "testGroupAddDuplicatedMember", "duplicatedUid")
	assert.Error(t, err)
	assert.Equal(t, constants.ErrMemberAlreadyExists, err)
}

func TestGroupContainsMember(t *testing.T) {
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
		{"backend_nouid", false, "", constants.ErrEmptyUID},
	}

	err := GroupCreate(ctx, "testGroupContainsMember")
	assert.NoError(t, err)

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err = GroupAddMember(ctx, "testGroupContainsMember", s.UID())
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			b, err := GroupContainsMember(ctx, "testGroupContainsMember", table.UID)
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

func TestRemove(t *testing.T) {
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

	err := GroupCreate(ctx, "testRemove")
	assert.NoError(t, err)

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err = GroupAddMember(ctx, "testRemove", s.UID())
			assert.NoError(t, err)
			err = GroupRemoveMember(ctx, "testRemove", s.UID())
			assert.NoError(t, err)
			res, err := GroupContainsMember(ctx, "testRemove", table.UID)
			assert.NoError(t, err)
			assert.False(t, res)
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	tables := []struct {
		name     string
		frontend bool
		UID      string
		err      error
	}{
		{"frontend_uid", true, "leaveSomeuid2", nil},
		{"backend_uid", false, "leaveOla2", nil},
	}

	err := GroupCreate(ctx, "testDeleteSufix")
	assert.NoError(t, err)

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			err = GroupCreate(ctx, "testDelete")
			assert.NoError(t, err)
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err = GroupAddMember(ctx, "testDeleteSufix", s.UID())
			assert.NoError(t, err)
			err = GroupAddMember(ctx, "testDelete", s.UID())
			assert.NoError(t, err)
			err = GroupDelete(ctx, "testDelete")
			assert.NoError(t, err)

			res, err := GroupContainsMember(ctx, "testDeleteSufix", table.UID)
			assert.NoError(t, err)
			assert.True(t, res)
			_, err = GroupContainsMember(ctx, "testDelete", table.UID)
			assert.Error(t, err)
			assert.Equal(t, constants.ErrGroupNotFound, err)
		})
	}
}

func TestRemoveAll(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	tables := []struct {
		name     string
		frontend bool
		UID      string
		err      error
	}{
		{"frontend_uid", true, "removeSomeuid2", nil},
		{"backend_uid", false, "removeOla2", nil},
	}

	err := GroupCreate(ctx, "testRemoveAllSufix")
	assert.NoError(t, err)
	err = GroupCreate(ctx, "testRemoveAll")
	assert.NoError(t, err)

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err = GroupAddMember(ctx, "testRemoveAllSufix", s.UID())
			assert.NoError(t, err)
			err = GroupAddMember(ctx, "testRemoveAll", s.UID())
			assert.NoError(t, err)
			err = GroupRemoveAll(ctx, "testRemoveAll")
			assert.NoError(t, err)

			res, err := GroupContainsMember(ctx, "testRemoveAllSufix", table.UID)
			assert.NoError(t, err)
			assert.True(t, res)
			res, err = GroupContainsMember(ctx, "testRemoveAll", table.UID)
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

	err := GroupCreate(ctx, "testCount")
	assert.NoError(t, err)

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
			s := session.New(mockNetworkEntity, table.frontend, table.UID)
			err = GroupAddMember(ctx, "testCount", s.UID())
			assert.NoError(t, err)
			res, err := GroupCountMembers(ctx, "testCount")
			assert.NoError(t, err)
			assert.Equal(t, 1, res)

			err = GroupRemoveAll(ctx, "testCount")
			assert.NoError(t, err)
		})
	}
}

func TestMembers(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	err := GroupCreate(ctx, "testGroupMembers")
	assert.NoError(t, err)
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true, "someid1")
	s2 := session.New(mockNetworkEntity, true, "someid2")
	err = GroupAddMember(ctx, "testGroupMembers", s1.UID())
	assert.NoError(t, err)
	err = GroupAddMember(ctx, "testGroupMembers", s2.UID())
	assert.NoError(t, err)

	res, err := GroupMembers(ctx, "testGroupMembers")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"someid1", "someid2"}, res)
}

func TestBroadcast(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	err := GroupCreate(ctx, "testBroadcast")
	assert.NoError(t, err)
	mockNetworkEntity := mocks.NewMockNetworkEntity(ctrl)
	s1 := session.New(mockNetworkEntity, true)
	s2 := session.New(mockNetworkEntity, true)
	err = s1.Bind(ctx, strconv.Itoa(int(s1.ID())))
	assert.NoError(t, err)
	err = s2.Bind(ctx, strconv.Itoa(int(s2.ID())))
	assert.NoError(t, err)
	err = GroupAddMember(ctx, "testBroadcast", s1.UID())
	assert.NoError(t, err)
	err = GroupAddMember(ctx, "testBroadcast", s2.UID())
	assert.NoError(t, err)
	route := "some.route.bla"
	data := []byte("hellow")
	mockNetworkEntity.EXPECT().Push(route, data).Times(2)
	err = GroupBroadcast(ctx, "testtype", "testBroadcast", route, data)
	assert.NoError(t, err)
}
