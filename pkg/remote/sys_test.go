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

package remote

import (
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v3/pkg/constants"
	"github.com/topfreegames/pitaya/v3/pkg/protos"
	"github.com/topfreegames/pitaya/v3/pkg/session/mocks"
)

func TestBindSession(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	id := int64(1)
	uid := uuid.New().String()
	d, err := json.Marshal(map[string]interface{}{
		"hello": "test",
	})
	assert.NoError(t, err)

	ss := mocks.NewMockSession(ctrl)

	sessionPool := mocks.NewMockSessionPool(ctrl)
	sessionPool.EXPECT().GetSessionByID(id).Return(ss).Times(1)

	s := NewSys(sessionPool)
	data := &protos.Session{
		Id:   id,
		Uid:  uid,
		Data: d,
	}

	ss.EXPECT().Bind(nil, uid).Times(1)

	res, err := s.BindSession(nil, data)
	assert.NoError(t, err)
	assert.Equal(t, []byte("ack"), res.Data)
}

func TestBindSessionShouldErrorIfNotExists(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	uid := uuid.New().String()
	d, err := json.Marshal(map[string]interface{}{
		"hello": "test",
	})
	assert.NoError(t, err)

	sessionPool := mocks.NewMockSessionPool(ctrl)
	sessionPool.EXPECT().GetSessionByID(gomock.Any()).Return(nil)

	s := NewSys(sessionPool)
	data := &protos.Session{
		Id:   133,
		Uid:  uid,
		Data: d,
	}
	_, err = s.BindSession(nil, data)
	assert.EqualError(t, constants.ErrSessionNotFound, err.Error())
}

func TestBindSessionShouldErrorIfAlreadyBound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	id := int64(1)
	uid := uuid.New().String()
	d, err := json.Marshal(map[string]interface{}{
		"hello": "test",
	})
	assert.NoError(t, err)
	data := &protos.Session{
		Id:   id,
		Uid:  uid,
		Data: d,
	}

	ss := mocks.NewMockSession(ctrl)
	ss.EXPECT().ID().Return(id).Times(1)
	ss.EXPECT().Bind(nil, uid).Return(nil).Times(1)
	ss.EXPECT().Bind(nil, uid).Return(constants.ErrSessionAlreadyBound)

	sessionPool := mocks.NewMockSessionPool(ctrl)
	sessionPool.EXPECT().GetSessionByID(ss.ID()).Return(ss).Times(2)

	s := NewSys(sessionPool)
	res, err := s.BindSession(nil, data)
	assert.NoError(t, err)
	assert.Equal(t, []byte("ack"), res.Data)

	_, err = s.BindSession(nil, data)
	assert.EqualError(t, constants.ErrSessionAlreadyBound, err.Error())
}

func TestPushSession(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	id := int64(1)
	uid := uuid.New().String()
	d, err := json.Marshal(map[string]interface{}{
		"hello":   "test",
		"hello22": 2,
	})
	data := &protos.Session{
		Id:   id,
		Uid:  uid,
		Data: d,
	}
	assert.NoError(t, err)

	ss := mocks.NewMockSession(ctrl)
	ss.EXPECT().SetDataEncoded(d).Times(1)

	sessionPool := mocks.NewMockSessionPool(ctrl)
	sessionPool.EXPECT().GetSessionByID(id).Return(ss).Times(1)

	s := NewSys(sessionPool)
	res, err := s.PushSession(nil, data)
	assert.NoError(t, err)
	assert.Equal(t, []byte("ack"), res.Data)
}

func TestPushSessionShouldFailIfSessionDoesntExists(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	uid := uuid.New().String()
	d, err := json.Marshal(map[string]interface{}{
		"hello":   "test",
		"hello22": 2,
	})
	assert.NoError(t, err)
	data := &protos.Session{
		Id:   343,
		Uid:  uid,
		Data: d,
	}

	sessionPool := mocks.NewMockSessionPool(ctrl)
	sessionPool.EXPECT().GetSessionByID(data.Id).Return(nil).Times(1)

	s := NewSys(sessionPool)
	_, err = s.PushSession(nil, data)
	assert.EqualError(t, constants.ErrSessionNotFound, err.Error())
}

func TestKick(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	uid := uuid.New().String()

	ss := mocks.NewMockSession(ctrl)
	ss.EXPECT().Kick(nil).Return(nil)

	sessionPool := mocks.NewMockSessionPool(ctrl)
	sessionPool.EXPECT().GetSessionByUID(uid).Return(ss).Times(1)

	s := NewSys(sessionPool)

	res, err := s.Kick(nil, &protos.KickMsg{UserId: uid})
	assert.NoError(t, err)
	assert.True(t, res.Kicked)
}

func TestKickSessionShouldFailIfSessionDoesntExists(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	uid := uuid.New().String()
	sessionPool := mocks.NewMockSessionPool(ctrl)
	sessionPool.EXPECT().GetSessionByUID(uid).Return(nil).Times(1)

	s := NewSys(sessionPool)
	_, err := s.Kick(nil, &protos.KickMsg{UserId: uid})
	assert.EqualError(t, constants.ErrSessionNotFound, err.Error())
}
