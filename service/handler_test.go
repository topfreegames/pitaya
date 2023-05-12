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

package service

import (
	"context"
	encjson "encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	agentmocks "github.com/topfreegames/pitaya/v2/agent/mocks"
	"github.com/topfreegames/pitaya/v2/cluster"
	"github.com/topfreegames/pitaya/v2/component"
	"github.com/topfreegames/pitaya/v2/conn/codec"
	"github.com/topfreegames/pitaya/v2/conn/message"
	"github.com/topfreegames/pitaya/v2/conn/packet"
	"github.com/topfreegames/pitaya/v2/constants"
	pcontext "github.com/topfreegames/pitaya/v2/context"
	"github.com/topfreegames/pitaya/v2/helpers"
	"github.com/topfreegames/pitaya/v2/metrics"
	metricsmocks "github.com/topfreegames/pitaya/v2/metrics/mocks"
	connmock "github.com/topfreegames/pitaya/v2/mocks"
	"github.com/topfreegames/pitaya/v2/pipeline"
	"github.com/topfreegames/pitaya/v2/protos"
	"github.com/topfreegames/pitaya/v2/route"
	"github.com/topfreegames/pitaya/v2/serialize/json"
	serializemocks "github.com/topfreegames/pitaya/v2/serialize/mocks"
	"github.com/topfreegames/pitaya/v2/session"
	"github.com/topfreegames/pitaya/v2/session/mocks"
)

var (
	once sync.Once
)

type mockAddr struct{}

func (m *mockAddr) Network() string { return "" }
func (m *mockAddr) String() string  { return "remote-string" }

type MyComp struct {
	component.Base
}

func (m *MyComp) Init()                        {}
func (m *MyComp) Shutdown()                    {}
func (m *MyComp) Handler1(ctx context.Context) {}
func (m *MyComp) Handler2(ctx context.Context, b []byte) ([]byte, error) {
	return nil, nil
}
func (m *MyComp) HandlerRawRaw(ctx context.Context, b []byte) ([]byte, error) {
	return b, nil
}

type NoHandlerRemoteComp struct {
	component.Base
}

func (m *NoHandlerRemoteComp) Init()     {}
func (m *NoHandlerRemoteComp) Shutdown() {}

func TestNewHandlerService(t *testing.T) {
	packetDecoder := codec.NewPomeloPacketDecoder()
	serializer := json.NewSerializer()
	sv := &cluster.Server{}
	remoteSvc := &RemoteService{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)
	mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}
	mockAgentFactory := agentmocks.NewMockAgentFactory(ctrl)
	handlerHooks := pipeline.NewHandlerHooks()
	handlerPool := NewHandlerPool()
	svc := NewHandlerService(
		packetDecoder,
		serializer,
		9, 8,
		sv,
		remoteSvc,
		mockAgentFactory,
		mockMetricsReporters,
		handlerHooks,
		handlerPool,
	)

	assert.NotNil(t, svc)
	assert.Equal(t, packetDecoder, svc.decoder)
	assert.Equal(t, serializer, svc.serializer)
	assert.Equal(t, mockMetricsReporters, svc.metricsReporters)
	assert.Equal(t, sv, svc.server)
	assert.Equal(t, remoteSvc, svc.remoteService)
	assert.Equal(t, mockAgentFactory, svc.agentFactory)
	assert.NotNil(t, svc.chLocalProcess)
	assert.NotNil(t, svc.chRemoteProcess)
	assert.Equal(t, handlerHooks, svc.handlerHooks)
	assert.Equal(t, handlerPool, svc.handlerPool)
}

func TestHandlerServiceRegister(t *testing.T) {
	handlerPool := NewHandlerPool()
	svc := NewHandlerService(nil, nil, 0, 0, nil, nil, nil, nil, nil, handlerPool)
	err := svc.Register(&MyComp{}, []component.Option{})
	assert.NoError(t, err)
	assert.Len(t, svc.services, 1)
	val, ok := svc.services["MyComp"]
	assert.True(t, ok)
	assert.NotNil(t, val)
	val2, ok := handlerPool.GetHandlers()["MyComp.Handler1"]
	assert.True(t, ok)
	assert.NotNil(t, val2)
	val2, ok = handlerPool.GetHandlers()["MyComp.Handler2"]
	assert.True(t, ok)
	assert.NotNil(t, val2)
	val2, ok = handlerPool.GetHandlers()["MyComp.HandlerRawRaw"]
	assert.True(t, ok)
	assert.NotNil(t, val2)
}

func TestHandlerServiceRegisterFailsIfRegisterTwice(t *testing.T) {
	handlerPool := NewHandlerPool()
	svc := NewHandlerService(nil, nil, 0, 0, nil, nil, nil, nil, nil, handlerPool)
	err := svc.Register(&MyComp{}, []component.Option{})
	assert.NoError(t, err)
	err = svc.Register(&MyComp{}, []component.Option{})
	assert.Contains(t, err.Error(), "handler: service already defined")
}

func TestHandlerServiceRegisterFailsIfNoHandlerMethods(t *testing.T) {
	handlerPool := NewHandlerPool()
	svc := NewHandlerService(nil, nil, 0, 0, nil, nil, nil, nil, nil, handlerPool)
	err := svc.Register(&NoHandlerRemoteComp{}, []component.Option{})
	assert.Equal(t, errors.New("type NoHandlerRemoteComp has no exported methods of handler type"), err)
}

func TestHandlerServiceProcessMessage(t *testing.T) {
	tables := []struct {
		name  string
		msg   *message.Message
		err   interface{}
		local bool
	}{
		{"failed_decode", &message.Message{ID: 1, Route: "k.k.k.k"}, &protos.Error{Msg: "invalid route", Code: "PIT-400"}, false},
		{"local_process", &message.Message{ID: 1, Route: "k.k"}, nil, true},
		{"remote_process", &message.Message{ID: 1, Route: "k.k.k"}, nil, false},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sv := &cluster.Server{}
			handlerPool := NewHandlerPool()
			svc := NewHandlerService(nil, nil, 1, 1, sv, &RemoteService{}, nil, nil, nil, handlerPool)

			mockSession := mocks.NewMockSession(ctrl)
			mockSession.EXPECT().UID().Return("uid").Times(1)
			mockAgent := agentmocks.NewMockAgent(ctrl)
			mockAgent.EXPECT().GetSession().Return(mockSession).Times(2)

			if table.err != nil {
				mockAgent.EXPECT().AnswerWithError(gomock.Any(), table.msg.ID, gomock.Any()).Times(1)
			}

			svc.processMessage(mockAgent, table.msg)

			if table.err == nil {
				var recvMsg unhandledMessage
				if table.err == nil && table.local {
					recvMsg = helpers.ShouldEventuallyReceive(t, svc.chLocalProcess).(unhandledMessage)
				} else if table.err == nil {
					recvMsg = helpers.ShouldEventuallyReceive(t, svc.chRemoteProcess).(unhandledMessage)
				}
				assert.Equal(t, table.msg, recvMsg.msg)
				assert.NotNil(t, pcontext.GetFromPropagateCtx(recvMsg.ctx, constants.StartTimeKey))
				assert.Equal(t, table.msg.Route, pcontext.GetFromPropagateCtx(recvMsg.ctx, constants.RouteKey))
			}
		})
	}
}

func TestHandlerServiceLocalProcess(t *testing.T) {
	tObj := &MyComp{}
	m, ok := reflect.TypeOf(tObj).MethodByName("HandlerRawRaw")
	assert.True(t, ok)
	assert.NotNil(t, m)
	rt := route.NewRoute("", uuid.New().String(), uuid.New().String())
	handlerPool := NewHandlerPool()
	handlerPool.handlers[rt.Short()] = &component.Handler{Receiver: reflect.ValueOf(tObj), Method: m, Type: m.Type.In(2), IsRawArg: true}

	tables := []struct {
		name string
		msg  *message.Message
		rt   *route.Route
		err  interface{}
	}{
		{"process_handler_msg_err", &message.Message{}, route.NewRoute("bla", "bla", "bla"), &protos.Error{Msg: "pitaya/handler: bla.bla.bla not found", Code: "PIT-404"}},
		{"success", &message.Message{ID: 1, Data: []byte(`["ok"]`)}, rt, nil},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := mocks.NewMockSession(ctrl)
			mockSession.EXPECT().UID().Return("uid").Times(1)

			mockAgent := agentmocks.NewMockAgent(ctrl)
			mockAgent.EXPECT().GetSession().Return(mockSession).AnyTimes()

			svc := NewHandlerService(nil, nil, 1, 1, nil, nil, nil, nil, pipeline.NewHandlerHooks(), handlerPool)

			ctx := context.Background()

			if table.err != nil {
				mockAgent.EXPECT().AnswerWithError(gomock.Any(), table.msg.ID, gomock.Any())
			} else {
				mockSession.EXPECT().ResponseMID(ctx, table.msg.ID, table.msg.Data, gomock.Any()).Return(nil).Times(1)
				mockSession.EXPECT().ID().Return(int64(1)).Times(1)
			}

			svc.localProcess(ctx, mockAgent, table.rt, table.msg)
		})
	}
}

func TestHandlerServiceProcessPacketHandshake(t *testing.T) {
	tables := []struct {
		name         string
		packet       *packet.Packet
		socketStatus int32
		validator    func(data *session.HandshakeData) error
		errStr       string
	}{
		{"invalid_handshake_data", &packet.Packet{Type: packet.Handshake, Data: []byte("asiodjasd")}, constants.StatusClosed, nil, "Invalid handshake data"},
		{"validator_error", &packet.Packet{Type: packet.Handshake, Data: []byte(`{"sys":{"platform":"mac"}}`)}, constants.StatusClosed, func(data *session.HandshakeData) error { return errors.New("validation failed") }, "Handshake validation failed"},
		{"valid_handshake_data", &packet.Packet{Type: packet.Handshake, Data: []byte(`{"sys":{"platform":"mac"}}`)}, constants.StatusHandshake, func(data *session.HandshakeData) error { return nil }, ""},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := mocks.NewMockSession(ctrl)
			mockSession.EXPECT().ID().Return(int64(1)).Times(1)

			mockAgent := agentmocks.NewMockAgent(ctrl)
			mockAgent.EXPECT().GetSession().Return(mockSession).Times(1)
			mockAgent.EXPECT().RemoteAddr().Return(&mockAddr{})
			mockAgent.EXPECT().SetStatus(table.socketStatus).Times(1)
			mockAgent.EXPECT().SendHandshakeResponse().Return(nil).Times(1)

			if table.errStr == "" {
				handshakeData := &session.HandshakeData{}
				_ = encjson.Unmarshal(table.packet.Data, handshakeData)
				mockAgent.EXPECT().GetSession().Return(mockSession).Times(3)
				mockAgent.EXPECT().IPVersion().Return(constants.IPv4).Times(1)
				mockSession.EXPECT().GetHandshakeValidators().Return([]func(data *session.HandshakeData) error{table.validator}).Times(1)
				mockSession.EXPECT().SetHandshakeData(handshakeData).Times(1)
				mockSession.EXPECT().Set(constants.IPVersionKey, constants.IPv4).Times(1)
				mockAgent.EXPECT().SetLastAt().Times(1)
			} else {
				mockAgent.EXPECT().GetSession().Return(mockSession).Times(1)
				mockSession.EXPECT().ID().Return(int64(1)).Times(1)
				if table.validator != nil {
					mockAgent.EXPECT().GetSession().Return(mockSession).Times(1)
					mockSession.EXPECT().GetHandshakeValidators().Return([]func(data *session.HandshakeData) error{table.validator}).Times(1)
				}
			}

			handlerPool := NewHandlerPool()
			svc := NewHandlerService(nil, nil, 1, 1, nil, nil, nil, nil, pipeline.NewHandlerHooks(), handlerPool)
			err := svc.processPacket(mockAgent, table.packet)
			if table.errStr == "" {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), table.errStr)
			}
		})
	}
}

func TestHandlerServiceProcessPacketHandshakeAck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := mocks.NewMockSession(ctrl)
	mockSession.EXPECT().ID().Return(int64(1)).Times(1)

	handlerPool := NewHandlerPool()
	svc := NewHandlerService(nil, nil, 1, 1, nil, nil, nil, nil, nil, handlerPool)

	mockAgent := agentmocks.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetSession().Return(mockSession).Times(1)
	mockAgent.EXPECT().SetStatus(constants.StatusWorking).Times(1)
	mockAgent.EXPECT().RemoteAddr().Return(&mockAddr{})
	mockAgent.EXPECT().SetLastAt()

	err := svc.processPacket(mockAgent, &packet.Packet{Type: packet.HandshakeAck})
	assert.NoError(t, err)
}

func TestHandlerServiceProcessPacketHeartbeat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAgent := agentmocks.NewMockAgent(ctrl)
	mockAgent.EXPECT().SetLastAt()

	handlerPool := NewHandlerPool()
	svc := NewHandlerService(nil, nil, 1, 1, nil, nil, nil, nil, nil, handlerPool)

	err := svc.processPacket(mockAgent, &packet.Packet{Type: packet.Heartbeat})
	assert.NoError(t, err)
}

func TestHandlerServiceProcessPacketData(t *testing.T) {
	msgID := uint(1)
	msg := &message.Message{Type: message.Request, ID: msgID, Data: []byte("ok")}
	messageEncoder := message.NewMessagesEncoder(false)
	encodedMsg, err := messageEncoder.Encode(msg)
	assert.NoError(t, err)
	tables := []struct {
		name         string
		packet       *packet.Packet
		socketStatus int32
		errStr       string
	}{
		{"not_acked_socket", &packet.Packet{Type: packet.Data, Data: []byte("ok")}, constants.StatusStart, "not yet ACK"},
		{"failed_decode", &packet.Packet{Type: packet.Data, Data: []byte("ok")}, constants.StatusWorking, "wrong message type"},
		{"success", &packet.Packet{Type: packet.Data, Data: encodedMsg}, constants.StatusWorking, ""},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := mocks.NewMockSession(ctrl)

			mockAgent := agentmocks.NewMockAgent(ctrl)

			mockAgent.EXPECT().GetStatus().Return(table.socketStatus)
			if table.socketStatus < constants.StatusWorking {
				mockAgent.EXPECT().RemoteAddr().Return(&mockAddr{})
			} else {
				if table.errStr == "" {
					mockAgent.EXPECT().GetSession().Return(mockSession).Times(2)
					mockSession.EXPECT().UID().Return("uid").Times(1)

					mockAgent.EXPECT().AnswerWithError(gomock.Any(), msgID, gomock.Any()).Times(1)
					mockAgent.EXPECT().SetLastAt().Times(1)
				}
			}

			handlerPool := NewHandlerPool()
			svc := NewHandlerService(nil, nil, 1, 1, &cluster.Server{}, nil, nil, nil, nil, handlerPool)
			err := svc.processPacket(mockAgent, table.packet)
			if table.errStr != "" {
				assert.Contains(t, err.Error(), table.errStr)
			}
		})
	}
}

func TestHandlerServiceHandle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	packetEncoder := codec.NewPomeloPacketEncoder()
	packetDecoder := codec.NewPomeloPacketDecoder()
	handshakeBuffer := `{"sys":{"platform":"mac","libVersion":"0.3.5-release","clientBuildNumber":"20","clientVersion":"2.1"},"user":{"age":30}}`
	bbb, err := packetEncoder.Encode(packet.Handshake, []byte(handshakeBuffer))
	assert.NoError(t, err)

	mockSerializer := serializemocks.NewMockSerializer(ctrl)

	mockConn := connmock.NewMockPlayerConn(ctrl)

	mockAgent := agentmocks.NewMockAgent(ctrl)
	mockAgentFactory := agentmocks.NewMockAgentFactory(ctrl)
	mockAgentFactory.EXPECT().CreateAgent(mockConn).Return(mockAgent).Times(1)

	var wg sync.WaitGroup
	wg.Add(4)
	defer wg.Wait()

	mockAgent.EXPECT().Handle().Do(func() {
		wg.Done()
	})

	mockAgent.EXPECT().SendHandshakeResponse().Return(nil)

	mockSession := mocks.NewMockSession(ctrl)
	mockSession.EXPECT().SetHandshakeData(gomock.Any()).Times(1)
	mockSession.EXPECT().UID().Return("uid").Times(1)
	mockSession.EXPECT().ID().Return(int64(1)).Times(2)
	mockSession.EXPECT().Set(constants.IPVersionKey, constants.IPv4)
	mockSession.EXPECT().Close()

	mockAgent.EXPECT().String().Return("")
	mockAgent.EXPECT().SetStatus(constants.StatusHandshake)
	mockAgent.EXPECT().GetSession().Return(mockSession).Times(6)
	mockAgent.EXPECT().IPVersion().Return(constants.IPv4)
	mockAgent.EXPECT().RemoteAddr().Return(&mockAddr{}).AnyTimes()
	mockAgent.EXPECT().SetLastAt().Do(func() {
		wg.Done()
	})

	firstCall := mockConn.EXPECT().GetNextMessage().Return(bbb, nil).Do(func() {
		wg.Done()
	})

	mockConn.EXPECT().GetNextMessage().Return(nil, errors.New("die")).Do(func() {
		wg.Done()
	}).After(firstCall)

	mockConn.EXPECT().Close().MaxTimes(1)

	handlerPool := NewHandlerPool()
	svc := NewHandlerService(packetDecoder, mockSerializer, 1, 1, nil, nil, mockAgentFactory, nil, pipeline.NewHandlerHooks(), handlerPool)
	svc.Handle(mockConn)
}
