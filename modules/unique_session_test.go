package modules

import (
	"context"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v2/cluster"
	clustermocks "github.com/topfreegames/pitaya/v2/cluster/mocks"
	"github.com/topfreegames/pitaya/v2/session"
	sessionmocks "github.com/topfreegames/pitaya/v2/session/mocks"
)

func TestUniqueSessionBind_ShouldKickOldSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := cluster.Server{}
	rpcClientMock := clustermocks.NewMockRPCClient(ctrl)
	sessionPoolMock := sessionmocks.NewMockSessionPool(ctrl)
	uniqueSession := NewUniqueSession(&server, nil, rpcClientMock, sessionPoolMock)

	uid := "test-uid"

	newSessionMock := sessionmocks.NewMockSession(ctrl)
	newSessionMock.EXPECT().UID().Return(uid).AnyTimes()
	newSessionMock.EXPECT().ID().Return(int64(2)).AnyTimes()

	oldSessionMock := sessionmocks.NewMockSession(ctrl)
	oldSessionMock.EXPECT().ID().Return(int64(1)).AnyTimes()

	sessionPoolMock.EXPECT().GetSessionByUID(uid).Return(oldSessionMock)
	oldSessionMock.EXPECT().Kick(gomock.Any()).Return(nil)

	rpcClientMock.EXPECT().BroadcastSessionBind(uid).Return(nil)

	var onSessionBindFunc func(ctx context.Context, s session.Session) error

	sessionPoolMock.EXPECT().OnSessionBind(gomock.Any()).DoAndReturn(func(f func(ctx context.Context, s session.Session) error) {
		onSessionBindFunc = f
	})

	uniqueSession.Init()

	err := onSessionBindFunc(context.Background(), newSessionMock)

	assert.NoError(t, err)
}

func TestUniqueSessionBind_ShouldCloseIfKickReturnsUnknownError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := cluster.Server{}
	rpcServerMock := clustermocks.NewMockRPCServer(ctrl)
	rpcClientMock := clustermocks.NewMockRPCClient(ctrl)
	sessionPoolMock := sessionmocks.NewMockSessionPool(ctrl)
	uniqueSession := NewUniqueSession(&server, rpcServerMock, rpcClientMock, sessionPoolMock)

	uid := "test-uid"

	newSessionMock := sessionmocks.NewMockSession(ctrl)
	newSessionMock.EXPECT().UID().Return(uid).AnyTimes()
	newSessionMock.EXPECT().ID().Return(int64(2)).AnyTimes()

	oldSessionMock := sessionmocks.NewMockSession(ctrl)
	oldSessionMock.EXPECT().ID().Return(int64(1)).AnyTimes()

	sessionPoolMock.EXPECT().GetSessionByUID(uid).Return(oldSessionMock)
	oldSessionMock.EXPECT().Kick(gomock.Any()).Return(assert.AnError)
	oldSessionMock.EXPECT().Close() // Should always close when kick fails

	rpcClientMock.EXPECT().BroadcastSessionBind(uid).Return(nil)

	var onSessionBindFunc func(ctx context.Context, s session.Session) error

	sessionPoolMock.EXPECT().OnSessionBind(gomock.Any()).DoAndReturn(func(f func(ctx context.Context, s session.Session) error) {
		onSessionBindFunc = f
	})

	uniqueSession.Init()

	err := onSessionBindFunc(context.Background(), newSessionMock)

	// Should succeed even if kick fails, because we force close the session
	assert.NoError(t, err)
}

func TestUniqueSessionBind_ShouldSucceedIfKickFailsWithKnownError(t *testing.T) {
	table := []struct {
		name string
		err  error
	}{
		{
			name: "deadline exceeded",
			err:  os.ErrDeadlineExceeded,
		},
		{
			name: "net closed",
			err:  net.ErrClosed,
		},
		{
			name: "broken pipe",
			err:  &net.OpError{Err: syscall.EPIPE},
		},
		{
			name: "connection reset",
			err:  &net.OpError{Err: syscall.ECONNRESET},
		},
	}

	for _, row := range table {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			server := cluster.Server{}
			rpcServerMock := clustermocks.NewMockRPCServer(ctrl)
			rpcClientMock := clustermocks.NewMockRPCClient(ctrl)
			sessionPoolMock := sessionmocks.NewMockSessionPool(ctrl)
			uniqueSession := NewUniqueSession(&server, rpcServerMock, rpcClientMock, sessionPoolMock)

			uid := "test-uid"

			newSessionMock := sessionmocks.NewMockSession(ctrl)
			newSessionMock.EXPECT().UID().Return(uid).AnyTimes()
			newSessionMock.EXPECT().ID().Return(int64(2)).AnyTimes()

			oldSessionMock := sessionmocks.NewMockSession(ctrl)
			oldSessionMock.EXPECT().ID().Return(int64(1)).AnyTimes()

			sessionPoolMock.EXPECT().GetSessionByUID(uid).Return(oldSessionMock)
			oldSessionMock.EXPECT().Kick(gomock.Any()).Return(row.err)
			oldSessionMock.EXPECT().Close()

			rpcClientMock.EXPECT().BroadcastSessionBind(uid).Return(nil)

			var onSessionBindFunc func(ctx context.Context, s session.Session) error

			sessionPoolMock.EXPECT().OnSessionBind(gomock.Any()).DoAndReturn(func(f func(ctx context.Context, s session.Session) error) {
				onSessionBindFunc = f
			})

			uniqueSession.Init()

			err := onSessionBindFunc(context.Background(), newSessionMock)

			assert.NoError(t, err)
		})
	}
}

func TestUniqueSessionBind_ShouldAlwaysCloseOnKickFailure(t *testing.T) {
	// This test verifies that regardless of the error type returned by Kick,
	// the unique session module always calls Close() on the old session
	table := []struct {
		name string
		err  error
	}{
		{
			name: "generic error",
			err:  assert.AnError,
		},
		{
			name: "network timeout",
			err:  os.ErrDeadlineExceeded,
		},
		{
			name: "connection closed",
			err:  net.ErrClosed,
		},
		{
			name: "broken pipe",
			err:  &net.OpError{Err: syscall.EPIPE},
		},
		{
			name: "connection reset by peer",
			err:  &net.OpError{Err: syscall.ECONNRESET},
		},
	}

	for _, row := range table {
		t.Run(row.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			server := cluster.Server{}
			rpcClientMock := clustermocks.NewMockRPCClient(ctrl)
			sessionPoolMock := sessionmocks.NewMockSessionPool(ctrl)
			uniqueSession := NewUniqueSession(&server, nil, rpcClientMock, sessionPoolMock)

			uid := "test-uid"

			newSessionMock := sessionmocks.NewMockSession(ctrl)
			newSessionMock.EXPECT().UID().Return(uid).AnyTimes()
			newSessionMock.EXPECT().ID().Return(int64(2)).AnyTimes()

			oldSessionMock := sessionmocks.NewMockSession(ctrl)
			oldSessionMock.EXPECT().ID().Return(int64(1)).AnyTimes()

			sessionPoolMock.EXPECT().GetSessionByUID(uid).Return(oldSessionMock)
			oldSessionMock.EXPECT().Kick(gomock.Any()).Return(row.err)
			// The key assertion: Close() should ALWAYS be called when Kick() fails
			oldSessionMock.EXPECT().Close()

			rpcClientMock.EXPECT().BroadcastSessionBind(uid).Return(nil)

			var onSessionBindFunc func(ctx context.Context, s session.Session) error

			sessionPoolMock.EXPECT().OnSessionBind(gomock.Any()).DoAndReturn(func(f func(ctx context.Context, s session.Session) error) {
				onSessionBindFunc = f
			})

			uniqueSession.Init()

			err := onSessionBindFunc(context.Background(), newSessionMock)

			// Should always succeed because we force close the session
			assert.NoError(t, err)
		})
	}
}
