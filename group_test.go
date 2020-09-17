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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v2/config"
	"github.com/topfreegames/pitaya/v2/constants"
	"github.com/topfreegames/pitaya/v2/session/mocks"
)

func createGroupTestApp() Pitaya {
	config := config.NewDefaultBuilderConfig()
	app := NewDefaultApp(true, "testtype", Cluster, map[string]string{}, config)
	return app
}

func TestCreateDuplicatedGroup(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	app := createGroupTestApp()
	err := app.GroupCreate(ctx, "testCreateDuplicatedGroup")
	assert.NoError(t, err)
	err = app.GroupCreate(ctx, "testCreateDuplicatedGroup")
	assert.Error(t, err)
	assert.Equal(t, constants.ErrGroupAlreadyExists, err)
}

func TestCreateGroup(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	app := createGroupTestApp()
	err := app.GroupCreate(ctx, "testCreateGroup")
	assert.NoError(t, err)
	count, err := app.GroupCountMembers(ctx, "testCreateGroup")
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	err = app.GroupRenewTTL(ctx, "testCreateGroup")
	assert.Error(t, err)
}

func TestCreateGroupWithTTL(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	app := createGroupTestApp()
	err := app.GroupCreateWithTTL(ctx, "testCreateGroupWithTTL", 10)
	assert.NoError(t, err)
	count, err := app.GroupCountMembers(ctx, "testCreateGroupWithTTL")
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	err = app.GroupRenewTTL(ctx, "testCreateGroupWithTTL")
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

	app := createGroupTestApp()
	err := app.GroupCreate(ctx, "testGroupAddMember")
	assert.NoError(t, err)

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := mocks.NewMockSession(ctrl)
			s.EXPECT().GetIsFrontend().Return(table.frontend).AnyTimes()
			s.EXPECT().UID().Return(table.UID).AnyTimes()
			err = app.GroupAddMember(ctx, "testGroupAddMember", s.UID())
			assert.Equal(t, table.err, err)
			if err == nil {
				_, err := app.GroupContainsMember(ctx, "testGroupAddMember", table.UID)
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
	app := createGroupTestApp()
	err := app.GroupCreate(ctx, "testGroupAddDuplicatedMember")
	assert.NoError(t, err)
	err = app.GroupAddMember(ctx, "testGroupAddDuplicatedMember", "duplicatedUid")
	assert.NoError(t, err)
	err = app.GroupAddMember(ctx, "testGroupAddDuplicatedMember", "duplicatedUid")
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

	app := createGroupTestApp()
	err := app.GroupCreate(ctx, "testGroupContainsMember")
	assert.NoError(t, err)

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			s := mocks.NewMockSession(ctrl)
			s.EXPECT().GetIsFrontend().Return(table.frontend).AnyTimes()
			s.EXPECT().UID().Return(table.UID).AnyTimes()
			err = app.GroupAddMember(ctx, "testGroupContainsMember", s.UID())
			if table.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			b, err := app.GroupContainsMember(ctx, "testGroupContainsMember", table.UID)
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

	app := createGroupTestApp()
	err := app.GroupCreate(ctx, "testRemove")
	assert.NoError(t, err)

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := mocks.NewMockSession(ctrl)
			s.EXPECT().GetIsFrontend().Return(table.frontend).AnyTimes()
			s.EXPECT().UID().Return(table.UID).AnyTimes()
			err = app.GroupAddMember(ctx, "testRemove", s.UID())
			assert.NoError(t, err)
			err = app.GroupRemoveMember(ctx, "testRemove", s.UID())
			assert.NoError(t, err)
			res, err := app.GroupContainsMember(ctx, "testRemove", table.UID)
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

	app := createGroupTestApp()
	err := app.GroupCreate(ctx, "testDeleteSufix")
	assert.NoError(t, err)

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			err = app.GroupCreate(ctx, "testDelete")
			assert.NoError(t, err)
			s := mocks.NewMockSession(ctrl)
			s.EXPECT().GetIsFrontend().Return(table.frontend).AnyTimes()
			s.EXPECT().UID().Return(table.UID).AnyTimes()
			err = app.GroupAddMember(ctx, "testDeleteSufix", s.UID())
			assert.NoError(t, err)
			err = app.GroupAddMember(ctx, "testDelete", s.UID())
			assert.NoError(t, err)
			err = app.GroupDelete(ctx, "testDelete")
			assert.NoError(t, err)

			res, err := app.GroupContainsMember(ctx, "testDeleteSufix", table.UID)
			assert.NoError(t, err)
			assert.True(t, res)
			_, err = app.GroupContainsMember(ctx, "testDelete", table.UID)
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

	app := createGroupTestApp()
	err := app.GroupCreate(ctx, "testRemoveAllSufix")
	assert.NoError(t, err)
	err = app.GroupCreate(ctx, "testRemoveAll")
	assert.NoError(t, err)

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := mocks.NewMockSession(ctrl)
			s.EXPECT().GetIsFrontend().Return(table.frontend).AnyTimes()
			s.EXPECT().UID().Return(table.UID).AnyTimes()
			err = app.GroupAddMember(ctx, "testRemoveAllSufix", s.UID())
			assert.NoError(t, err)
			err = app.GroupAddMember(ctx, "testRemoveAll", s.UID())
			assert.NoError(t, err)
			err = app.GroupRemoveAll(ctx, "testRemoveAll")
			assert.NoError(t, err)

			res, err := app.GroupContainsMember(ctx, "testRemoveAllSufix", table.UID)
			assert.NoError(t, err)
			assert.True(t, res)
			res, err = app.GroupContainsMember(ctx, "testRemoveAll", table.UID)
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

	app := createGroupTestApp()
	err := app.GroupCreate(ctx, "testCount")
	assert.NoError(t, err)

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := mocks.NewMockSession(ctrl)
			s.EXPECT().GetIsFrontend().Return(table.frontend).AnyTimes()
			s.EXPECT().UID().Return(table.UID).AnyTimes()
			err = app.GroupAddMember(ctx, "testCount", s.UID())
			assert.NoError(t, err)
			res, err := app.GroupCountMembers(ctx, "testCount")
			assert.NoError(t, err)
			assert.Equal(t, 1, res)

			err = app.GroupRemoveAll(ctx, "testCount")
			assert.NoError(t, err)
		})
	}
}

func TestMembers(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := createGroupTestApp()
	err := app.GroupCreate(ctx, "testGroupMembers")
	assert.NoError(t, err)
	s1 := mocks.NewMockSession(ctrl)
	s1.EXPECT().GetIsFrontend().Return(true).AnyTimes()
	s1.EXPECT().UID().Return("someid1").AnyTimes()

	s2 := mocks.NewMockSession(ctrl)
	s2.EXPECT().GetIsFrontend().Return(true).AnyTimes()
	s2.EXPECT().UID().Return("someid2").AnyTimes()

	err = app.GroupAddMember(ctx, "testGroupMembers", s1.UID())
	assert.NoError(t, err)
	err = app.GroupAddMember(ctx, "testGroupMembers", s2.UID())
	assert.NoError(t, err)

	res, err := app.GroupMembers(ctx, "testGroupMembers")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"someid1", "someid2"}, res)
}

func TestBroadcast(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	id1 := int64(1)
	id2 := int64(2)
	uid1 := "uid1"
	uid2 := "uid2"

	s1 := mocks.NewMockSession(ctrl)
	s1.EXPECT().GetIsFrontend().Return(true).AnyTimes()
	s1.EXPECT().UID().Return("someid1").AnyTimes()
	s1.EXPECT().ID().Return(id1).AnyTimes()

	s2 := mocks.NewMockSession(ctrl)
	s2.EXPECT().GetIsFrontend().Return(true).AnyTimes()
	s2.EXPECT().ID().Return(id2).AnyTimes()

	mockSessionPool := mocks.NewMockSessionPool(ctrl)
	mockSessionPool.EXPECT().GetSessionByUID(uid1).Return(s1).Times(1)
	mockSessionPool.EXPECT().GetSessionByUID(uid2).Return(s2).Times(1)

	config := config.NewDefaultBuilderConfig()
	builder := NewDefaultBuilder(true, "testtype", Cluster, map[string]string{}, config)
	builder.SessionPool = mockSessionPool
	app := builder.Build()

	err := app.GroupCreate(ctx, "testBroadcast")
	assert.NoError(t, err)

	err = app.GroupAddMember(ctx, "testBroadcast", uid1)
	assert.NoError(t, err)
	err = app.GroupAddMember(ctx, "testBroadcast", uid2)
	assert.NoError(t, err)
	route := "some.route.bla"
	data := []byte("hellow")
	s1.EXPECT().Push(route, data).Times(1)
	s2.EXPECT().Push(route, data).Times(1)
	err = app.GroupBroadcast(ctx, "testtype", "testBroadcast", route, data)
	assert.NoError(t, err)
}
