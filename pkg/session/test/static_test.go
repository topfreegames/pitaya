package test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/topfreegames/pitaya/v3/pkg/session"
	"github.com/topfreegames/pitaya/v3/pkg/session/mocks"
)

func TestStaticGetSessionByUID(t *testing.T) {
	createSessionMock := func(ctrl *gomock.Controller) session.Session {
		return mocks.NewMockSession(ctrl)
	}
	createNilSession := func(ctrl *gomock.Controller) session.Session {
		return mocks.NewMockSession(ctrl)
	}

	tables := []struct {
		name    string
		uid     string
		factory func(ctrl *gomock.Controller) session.Session
	}{
		{"Success", uuid.New().String(), createSessionMock},
		{"Error", uuid.New().String(), createNilSession},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			expectedSession := row.factory(ctrl)

			sessionPool := mocks.NewMockSessionPool(ctrl)
			sessionPool.EXPECT().GetSessionByUID(row.uid).Return(expectedSession)

			session.DefaultSessionPool = sessionPool
			session := session.GetSessionByUID(row.uid)
			require.Equal(t, expectedSession, session)
		})
	}
}

func TestStaticGetSessionByID(t *testing.T) {
	createSessionMock := func(ctrl *gomock.Controller) session.Session {
		return mocks.NewMockSession(ctrl)
	}
	createNilSession := func(ctrl *gomock.Controller) session.Session {
		return mocks.NewMockSession(ctrl)
	}

	tables := []struct {
		name    string
		id      int64
		factory func(ctrl *gomock.Controller) session.Session
	}{
		{"Success", 3, createSessionMock},
		{"Error", 3, createNilSession},
	}

	for _, row := range tables {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			expectedSession := row.factory(ctrl)

			sessionPool := mocks.NewMockSessionPool(ctrl)
			sessionPool.EXPECT().GetSessionByID(row.id).Return(expectedSession)

			session.DefaultSessionPool = sessionPool
			session := session.GetSessionByID(row.id)
			require.Equal(t, expectedSession, session)
		})
	}
}

func TestStaticOnSessionBind(t *testing.T) {
	ctrl := gomock.NewController(t)

	sessionPool := mocks.NewMockSessionPool(ctrl)
	sessionPool.EXPECT().OnSessionBind(nil)

	session.DefaultSessionPool = sessionPool
	session.OnSessionBind(nil)
}

func TestStaticOnAfterSessionBind(t *testing.T) {
	ctrl := gomock.NewController(t)

	sessionPool := mocks.NewMockSessionPool(ctrl)
	sessionPool.EXPECT().OnAfterSessionBind(nil)

	session.DefaultSessionPool = sessionPool
	session.OnAfterSessionBind(nil)
}

func TestStaticOnSessionClose(t *testing.T) {
	ctrl := gomock.NewController(t)

	sessionPool := mocks.NewMockSessionPool(ctrl)
	sessionPool.EXPECT().OnSessionClose(nil)

	session.DefaultSessionPool = sessionPool
	session.OnSessionClose(nil)
}

func TestStaticCloseAll(t *testing.T) {
	ctrl := gomock.NewController(t)

	sessionPool := mocks.NewMockSessionPool(ctrl)
	sessionPool.EXPECT().CloseAll()

	session.DefaultSessionPool = sessionPool
	session.CloseAll()
}
