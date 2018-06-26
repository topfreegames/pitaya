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
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/session/mocks"
)

func TestBindSession(t *testing.T) {
	t.Parallel()
	s := &Sys{}
	ss := session.New(nil, true)
	uid := uuid.New().String()
	d, err := json.Marshal(map[string]interface{}{
		"hello": "test",
	})
	assert.NoError(t, err)
	data := &protos.Session{
		Id:   ss.ID(),
		Uid:  uid,
		Data: d,
	}
	res, err := s.BindSession(nil, data)
	assert.NoError(t, err)
	assert.Equal(t, []byte("ack"), res.Data)
	assert.Equal(t, ss, session.GetSessionByUID(uid))
}

func TestBindSessionShouldErrorIfNotExists(t *testing.T) {
	t.Parallel()
	s := &Sys{}
	uid := uuid.New().String()
	d, err := json.Marshal(map[string]interface{}{
		"hello": "test",
	})
	assert.NoError(t, err)
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
	s := &Sys{}
	ss := session.New(nil, true)
	uid := uuid.New().String()
	d, err := json.Marshal(map[string]interface{}{
		"hello": "test",
	})
	assert.NoError(t, err)
	data := &protos.Session{
		Id:   ss.ID(),
		Uid:  uid,
		Data: d,
	}
	res, err := s.BindSession(nil, data)
	assert.NoError(t, err)
	assert.Equal(t, []byte("ack"), res.Data)
	assert.Equal(t, ss, session.GetSessionByUID(uid))
	_, err = s.BindSession(nil, data)
	assert.EqualError(t, constants.ErrSessionAlreadyBound, err.Error())
}

func TestPushSession(t *testing.T) {
	t.Parallel()
	s := &Sys{}
	ss := session.New(nil, true)
	uid := uuid.New().String()
	d, err := json.Marshal(map[string]interface{}{
		"hello":   "test",
		"hello22": 2,
	})
	assert.NoError(t, err)
	data := &protos.Session{
		Id:   ss.ID(),
		Uid:  uid,
		Data: d,
	}
	res, err := s.PushSession(nil, data)
	assert.NoError(t, err)
	assert.Equal(t, []byte("ack"), res.Data)
	assert.Equal(t, data.Data, ss.GetDataEncoded())
}

func TestPushSessionShouldFailIfSessionDoesntExists(t *testing.T) {
	t.Parallel()
	s := &Sys{}
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
	_, err = s.PushSession(nil, data)
	assert.EqualError(t, constants.ErrSessionNotFound, err.Error())
}

func TestKick(t *testing.T) {
	t.Parallel()
	s := &Sys{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockEntity := mocks.NewMockNetworkEntity(ctrl)
	ss := session.New(mockEntity, true)
	uid := uuid.New().String()
	d, err := json.Marshal(map[string]interface{}{
		"hello":   "test",
		"hello22": 2,
	})
	assert.NoError(t, err)
	data := &protos.Session{
		Id:   ss.ID(),
		Uid:  uid,
		Data: d,
	}
	_, err = s.BindSession(nil, data)
	assert.NoError(t, err)

	mockEntity.EXPECT().Kick(nil)
	mockEntity.EXPECT().Close()
	res, err := s.Kick(nil, &protos.KickMsg{UserId: uid})
	assert.NoError(t, err)
	assert.True(t, res.Kicked)
}

func TestKickSessionShouldFailIfSessionDoesntExists(t *testing.T) {
	t.Parallel()
	s := &Sys{}
	_, err := s.Kick(nil, &protos.KickMsg{UserId: uuid.New().String()})
	assert.EqualError(t, constants.ErrSessionNotFound, err.Error())
}
