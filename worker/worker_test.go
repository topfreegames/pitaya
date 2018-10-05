package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	workers "github.com/topfreegames/go-workers"
	"github.com/topfreegames/pitaya/worker/mocks"
)

type fakeProtoMessage struct {
	Field string
}

func (f *fakeProtoMessage) Reset() {}

func (f *fakeProtoMessage) String() string { return "" }

func (f *fakeProtoMessage) ProtoMessage() {}

func TestParsedRPCJob(t *testing.T) {
	t.Parallel()

	var (
		mockRPCJob *mocks.MockRPCJob
		testErr    = errors.New("error")
		route      = "server.svc.method"
		serverID   = "serverid"
		ctx        = context.Background()
	)

	tables := map[string]struct {
		arg    *workers.Msg
		mocks  func()
		assert func(func(*workers.Msg), *workers.Msg)
	}{
		"test_error_when_route_not_string": {
			arg: func() *workers.Msg {
				msg, err := workers.NewMsg(`{"args": {"route": 10}}`)
				assert.NoError(t, err)
				return msg
			}(),
			mocks: func() {},
			assert: func(f func(*workers.Msg), m *workers.Msg) {
				assert.Panics(t, func() { f(m) })
			},
		},
		"test_error_on_get_arg_reply": {
			arg: func() *workers.Msg {
				msg, err := workers.NewMsg(`{
					"args": { "route": "server.svc.method" }
				}`)
				assert.NoError(t, err)
				return msg
			}(),
			mocks: func() {
				mockRPCJob.EXPECT().
					GetArgReply(route).
					Return(nil, nil, testErr)
			},
			assert: func(f func(*workers.Msg), m *workers.Msg) {
				assert.Panics(t, func() { f(m) })
			},
		},
		"test_error_on_unmarshal_rpc_info": {
			arg: func() *workers.Msg {
				msg, err := workers.NewMsg(`{
					"args": { 
						"route": "server.svc.method",
						"arg": { "field": 10 }
					}
				}`)
				assert.NoError(t, err)
				return msg
			}(),
			mocks: func() {
				mockRPCJob.EXPECT().
					GetArgReply(route).
					Return(&fakeProtoMessage{}, &fakeProtoMessage{}, nil)
			},
			assert: func(f func(*workers.Msg), m *workers.Msg) {
				assert.Panics(t, func() { f(m) })
			},
		},
		"test_error_on_server_discovery": {
			arg: func() *workers.Msg {
				msg, err := workers.NewMsg(`{
					"args": { 
						"route": "server.svc.method",
						"arg": { "field": "string" },
						"metadata": { "stack": "a" }
					}
				}`)
				assert.NoError(t, err)
				return msg
			}(),
			mocks: func() {
				mockRPCJob.EXPECT().
					GetArgReply(route).
					Return(&fakeProtoMessage{}, &fakeProtoMessage{}, nil)
				mockRPCJob.EXPECT().
					ServerDiscovery(route, map[string]interface{}{"stack": "a"}).
					Return("", testErr)
			},
			assert: func(f func(*workers.Msg), m *workers.Msg) {
				assert.Panics(t, func() { f(m) })
			},
		},
		"test_error_on_rpc": {
			arg: func() *workers.Msg {
				msg, err := workers.NewMsg(`{
					"args": { 
						"route": "server.svc.method",
						"arg": { "field": "string" },
						"metadata": { "stack": "a" }
					}
				}`)
				assert.NoError(t, err)
				return msg
			}(),
			mocks: func() {
				mockRPCJob.EXPECT().
					GetArgReply(route).
					Return(&fakeProtoMessage{}, &fakeProtoMessage{}, nil)
				mockRPCJob.EXPECT().
					ServerDiscovery(route, map[string]interface{}{"stack": "a"}).
					Return(serverID, nil)
				mockRPCJob.EXPECT().
					RPC(ctx, serverID, route, &fakeProtoMessage{}, &fakeProtoMessage{Field: "string"}).
					Return(testErr)
			},
			assert: func(f func(*workers.Msg), m *workers.Msg) {
				assert.Panics(t, func() { f(m) })
			},
		},
		"test_execute_rpc": {
			arg: func() *workers.Msg {
				msg, err := workers.NewMsg(`{
					"args": { 
						"route": "server.svc.method",
						"arg": { "field": "string" },
						"metadata": { "stack": "a" }
					}
				}`)
				assert.NoError(t, err)
				return msg
			}(),
			mocks: func() {
				mockRPCJob.EXPECT().
					GetArgReply(route).
					Return(&fakeProtoMessage{}, &fakeProtoMessage{}, nil)
				mockRPCJob.EXPECT().
					ServerDiscovery(route, map[string]interface{}{"stack": "a"}).
					Return(serverID, nil)
				mockRPCJob.EXPECT().
					RPC(ctx, serverID, route, &fakeProtoMessage{}, &fakeProtoMessage{Field: "string"})
			},
			assert: func(f func(*workers.Msg), m *workers.Msg) {
				assert.NotPanics(t, func() { f(m) })
			},
		},
	}

	for name, table := range tables {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRPCJob = mocks.NewMockRPCJob(ctrl)
			table.mocks()

			worker := &Worker{}
			table.assert(worker.parsedRPCJob(mockRPCJob), table.arg)
		})
	}
}
