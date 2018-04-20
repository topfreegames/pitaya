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
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/agent"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/helpers"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/internal/packet"
	connmock "github.com/topfreegames/pitaya/mocks"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/serialize/json"
	serializemocks "github.com/topfreegames/pitaya/serialize/mocks"
	"github.com/topfreegames/pitaya/session"
)

type mockAddr struct{}

func (m *mockAddr) Network() string { return "" }
func (m *mockAddr) String() string  { return "remote-string" }

type MyComp struct {
	component.Base
}

func (m *MyComp) Init()                        {}
func (m *MyComp) Shutdown()                    {}
func (m *MyComp) Handler1(ss *session.Session) {}
func (m *MyComp) Handler2(ss *session.Session, b []byte) ([]byte, error) {
	return nil, nil
}
func (m *MyComp) HandlerRawRaw(ss *session.Session, b []byte) ([]byte, error) {
	return b, nil
}

type NoHandlerRemoteComp struct {
	component.Base
}

func (m *NoHandlerRemoteComp) Init()     {}
func (m *NoHandlerRemoteComp) Shutdown() {}

func TestNewHandlerService(t *testing.T) {
	dieChan := make(chan bool)
	packetDecoder := codec.NewPomeloPacketDecoder()
	packetEncoder := codec.NewPomeloPacketEncoder()
	serializer := json.NewSerializer()
	heartbeatTimeout := 1 * time.Second
	sv := &cluster.Server{}
	remoteSvc := &RemoteService{}
	svc := NewHandlerService(
		dieChan,
		packetDecoder,
		packetEncoder,
		serializer,
		heartbeatTimeout,
		10, 9, 8,
		sv,
		remoteSvc,
	)

	assert.NotNil(t, svc)
	assert.Equal(t, dieChan, svc.appDieChan)
	assert.Equal(t, packetDecoder, svc.decoder)
	assert.Equal(t, packetEncoder, svc.encoder)
	assert.Equal(t, serializer, svc.serializer)
	assert.Equal(t, heartbeatTimeout, svc.heartbeatTimeout)
	assert.Equal(t, 10, svc.messagesBufferSize)
	assert.Equal(t, sv, svc.server)
	assert.Equal(t, remoteSvc, svc.remoteService)
	assert.NotNil(t, svc.chLocalProcess)
	assert.NotNil(t, svc.chRemoteProcess)
}

func TestHandlerServiceRegister(t *testing.T) {
	svc := NewHandlerService(nil, nil, nil, nil, 0, 0, 0, 0, nil, nil)
	err := svc.Register(&MyComp{}, []component.Option{})
	assert.NoError(t, err)
	defer func() { handlers = make(map[string]*component.Handler, 0) }()
	assert.Len(t, svc.services, 1)
	val, ok := svc.services["MyComp"]
	assert.True(t, ok)
	assert.NotNil(t, val)
	assert.Len(t, handlers, 3)
	val2, ok := handlers["MyComp.Handler1"]
	assert.True(t, ok)
	assert.NotNil(t, val2)
	val2, ok = handlers["MyComp.Handler2"]
	assert.True(t, ok)
	assert.NotNil(t, val2)
	val2, ok = handlers["MyComp.HandlerRawRaw"]
	assert.True(t, ok)
	assert.NotNil(t, val2)
}

func TestHandlerServiceRegisterFailsIfRegisterTwice(t *testing.T) {
	svc := NewHandlerService(nil, nil, nil, nil, 0, 0, 0, 0, nil, nil)
	err := svc.Register(&MyComp{}, []component.Option{})
	assert.NoError(t, err)
	err = svc.Register(&MyComp{}, []component.Option{})
	assert.Contains(t, err.Error(), "handler: service already defined")
}

func TestHandlerServiceRegisterFailsIfNoHandlerMethods(t *testing.T) {
	svc := NewHandlerService(nil, nil, nil, nil, 0, 0, 0, 0, nil, nil)
	err := svc.Register(&NoHandlerRemoteComp{}, []component.Option{})
	assert.Equal(t, errors.New("type NoHandlerRemoteComp has no exported methods of suitable type"), err)
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
			packetEncoder := codec.NewPomeloPacketEncoder()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockConn := connmock.NewMockConn(ctrl)
			sv := &cluster.Server{}
			svc := NewHandlerService(nil, nil, nil, nil, 1*time.Second, 1, 1, 1, sv, &RemoteService{})

			if table.err != nil {
				mockSerializer.EXPECT().Marshal(table.err).Return([]byte("err"), nil)
			}

			ag := agent.NewAgent(mockConn, nil, packetEncoder, mockSerializer, 1*time.Second, 1, nil)
			svc.processMessage(ag, table.msg)

			if table.err == nil {
				var recvMsg unhandledMessage
				if table.err == nil && table.local {
					recvMsg = helpers.ShouldEventuallyReceive(t, svc.chLocalProcess).(unhandledMessage)
				} else if table.err == nil {
					recvMsg = helpers.ShouldEventuallyReceive(t, svc.chRemoteProcess).(unhandledMessage)
				}
				assert.Equal(t, table.msg, recvMsg.msg)
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
	handlers[rt.Short()] = &component.Handler{Receiver: reflect.ValueOf(tObj), Method: m, Type: m.Type.In(2), IsRawArg: true}

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

			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockConn := connmock.NewMockConn(ctrl)
			packetEncoder := codec.NewPomeloPacketEncoder()
			svc := NewHandlerService(nil, nil, nil, nil, 1*time.Second, 1, 1, 1, nil, nil)

			if table.err != nil {
				mockSerializer.EXPECT().Marshal(table.err)
			}
			ag := agent.NewAgent(mockConn, nil, packetEncoder, mockSerializer, 1*time.Second, 1, nil)
			svc.localProcess(ag, table.rt, table.msg)
		})
	}
}

func TestHandlerServiceProcessPacketHandshake(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockConn := connmock.NewMockConn(ctrl)
	packetEncoder := codec.NewPomeloPacketEncoder()
	svc := NewHandlerService(nil, nil, nil, nil, 1*time.Second, 1, 1, 1, nil, nil)

	mockConn.EXPECT().RemoteAddr().Return(&mockAddr{})
	mockConn.EXPECT().Write(gomock.Any()).Do(func(d []byte) {
		assert.Contains(t, string(d), "heartbeat")
	})
	ag := agent.NewAgent(mockConn, nil, packetEncoder, mockSerializer, 1*time.Second, 1, nil)
	err := svc.processPacket(ag, &packet.Packet{Type: packet.Handshake})
	assert.NoError(t, err)
	assert.Equal(t, constants.StatusHandshake, ag.GetStatus())
}

func TestHandlerServiceProcessPacketHandshakeAck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := connmock.NewMockConn(ctrl)
	packetEncoder := codec.NewPomeloPacketEncoder()
	svc := NewHandlerService(nil, nil, nil, nil, 1*time.Second, 1, 1, 1, nil, nil)

	mockConn.EXPECT().RemoteAddr().Return(&mockAddr{})
	ag := agent.NewAgent(mockConn, nil, packetEncoder, nil, 1*time.Second, 1, nil)
	err := svc.processPacket(ag, &packet.Packet{Type: packet.HandshakeAck})
	assert.NoError(t, err)
	assert.Equal(t, constants.StatusWorking, ag.GetStatus())
}

func TestHandlerServiceProcessPacketHeartbeat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := connmock.NewMockConn(ctrl)
	packetEncoder := codec.NewPomeloPacketEncoder()
	svc := NewHandlerService(nil, nil, nil, nil, 1*time.Second, 1, 1, 1, nil, nil)

	mockConn.EXPECT().RemoteAddr().Return(&mockAddr{})
	ag := agent.NewAgent(mockConn, nil, packetEncoder, nil, 1*time.Second, 1, nil)
	// wait to check if lastTime is updated. SORRY!
	time.Sleep(1 * time.Second)
	err := svc.processPacket(ag, &packet.Packet{Type: packet.Heartbeat})
	assert.NoError(t, err)
	assert.Contains(t, ag.String(), fmt.Sprintf("LastTime=%d", time.Now().Unix()))
}

func TestHandlerServiceProcessPacketData(t *testing.T) {
	msg := &message.Message{Type: message.Request, ID: 1, Data: []byte("ok")}
	encodedMsg, err := msg.Encode()
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

			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockConn := connmock.NewMockConn(ctrl)
			packetEncoder := codec.NewPomeloPacketEncoder()
			svc := NewHandlerService(nil, nil, nil, nil, 1*time.Second, 1, 1, 1, nil, nil)
			if table.socketStatus < constants.StatusWorking {
				mockConn.EXPECT().RemoteAddr().Return(&mockAddr{})
			}
			ag := agent.NewAgent(mockConn, nil, packetEncoder, mockSerializer, 1*time.Second, 1, nil)
			ag.SetStatus(table.socketStatus)

			if table.errStr == "" {
				mockSerializer.EXPECT().Marshal(&protos.Error{Code: "PIT-400", Msg: route.ErrRouteFieldCantEmpty.Error()})
			}
			err := svc.processPacket(ag, table.packet)
			if table.errStr != "" {
				assert.Contains(t, err.Error(), table.errStr)
			}
		})
	}
}

func TestHandlerServiceHandle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockConn := connmock.NewMockConn(ctrl)
	packetEncoder := codec.NewPomeloPacketEncoder()
	packetDecoder := codec.NewPomeloPacketDecoder()
	svc := NewHandlerService(nil, packetDecoder, packetEncoder, mockSerializer, 1*time.Second, 1, 1, 1, nil, nil)
	var wg sync.WaitGroup
	firstCall := mockConn.EXPECT().Read(gomock.Any()).Do(func(b []byte) {
		handshakeBuffer := `{
  "sys": {
    "type": "golang-tcp",
    "version": "0.0.1",
    "rsa": {}
  },
  "user": {}
};`
		bbb, err := packetEncoder.Encode(packet.Handshake, []byte(handshakeBuffer))
		for i, c := range bbb {
			b[i] = c
		}
		assert.NoError(t, err)
		wg.Done()
	}).Return(101, nil)

	mockConn.EXPECT().Read(gomock.Any()).Return(0, errors.New("die")).Do(func(b []byte) {
		wg.Done()
	}).After(firstCall)

	mockConn.EXPECT().RemoteAddr().Return(&mockAddr{}).AnyTimes()
	mockConn.EXPECT().Write(gomock.Any()).Do(func(d []byte) {
		assert.Contains(t, string(d), "heartbeat")
	})
	mockConn.EXPECT().Close().MaxTimes(1)

	wg.Add(2)
	go svc.Handle(mockConn)
	wg.Wait()
}
