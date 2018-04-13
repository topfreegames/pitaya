// Copyright (c) nano Author and TFG Co. All Rights Reserved.
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

package agent

import (
	"math/rand"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/helpers"
	codecmocks "github.com/topfreegames/pitaya/internal/codec/mocks"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/internal/packet"
	serializemocks "github.com/topfreegames/pitaya/serialize/mocks"
)

func TestNewAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	mockDecoder := codecmocks.NewMockPacketDecoder(ctrl)
	dieChan := make(chan bool)
	hbTime := time.Second

	conn, _ := net.Dial("tcp", "127.0.0.1:0")
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Not(gomock.Nil())).Do(
		func(typ packet.Type, d []byte) {
			// cannot compare inside the expect because they are equivalent but not equal
			assert.EqualValues(t, packet.Handshake, typ)
		})
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Nil()).Do(
		func(typ packet.Type, d []byte) {
			assert.EqualValues(t, packet.Heartbeat, typ)
		})

	ag := NewAgent(conn, mockDecoder, mockEncoder, mockSerializer, hbTime, 10, dieChan)
	assert.NotNil(t, ag)
	assert.IsType(t, make(chan struct{}), ag.chDie)
	assert.IsType(t, make(chan pendingMessage), ag.chSend)
	assert.IsType(t, make(chan struct{}), ag.chStopHeartbeat)
	assert.IsType(t, make(chan struct{}), ag.chStopWrite)
	assert.Equal(t, dieChan, ag.appDieChan)
	assert.Equal(t, 10, ag.messagesBufferSize)
	assert.Equal(t, conn, ag.conn)
	assert.Equal(t, mockDecoder, ag.decoder)
	assert.Equal(t, mockEncoder, ag.encoder)
	assert.Equal(t, hbTime, ag.heartbeatTimeout)
	assert.InDelta(t, time.Now().Unix(), ag.lastAt, 1)
	assert.Equal(t, mockSerializer, ag.serializer)
	assert.Equal(t, constants.StatusStart, ag.state)
	assert.NotNil(t, ag.Session)
	assert.True(t, ag.Session.IsFrontend)
	assert.Equal(t, reflect.ValueOf(ag.Session), ag.Srv)

	// second call should no call hdb encode
	ag = NewAgent(nil, nil, mockEncoder, nil, hbTime, 10, dieChan)
	assert.NotNil(t, ag)
}

func TestAgentSend(t *testing.T) {
	tables := []struct {
		name string
		err  error
	}{
		{"success", nil},
		{"failure", constants.ErrBrokenPipe},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
			// heartbeat and handshake if not set by another test
			// TODO: ugly ugly
			mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Not(gomock.Nil())).AnyTimes()
			mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Nil()).AnyTimes()
			mockDecoder := codecmocks.NewMockPacketDecoder(ctrl)
			dieChan := make(chan bool)
			hbTime := time.Second

			conn, _ := net.Dial("tcp", "127.0.0.1:0")
			ag := NewAgent(conn, mockDecoder, mockEncoder, mockSerializer, hbTime, 10, dieChan)
			assert.NotNil(t, ag)

			if table.err != nil {
				close(ag.chSend)
			}
			msg := pendingMessage{}
			err := ag.send(msg)
			assert.Equal(t, table.err, err)

			if table.err == nil {
				recvMsg := helpers.ShouldEventuallyReceive(t, ag.chSend).(pendingMessage)
				assert.Equal(t, msg, recvMsg)
			}
		})
	}
}

func TestAgentPushFailsIfClosedAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	// heartbeat and handshake if not set by another test
	// TODO: ugly ugly
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Not(gomock.Nil())).AnyTimes()
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Nil()).AnyTimes()

	ag := NewAgent(nil, nil, mockEncoder, nil, time.Second, 10, nil)
	assert.NotNil(t, ag)
	ag.state = constants.StatusClosed
	err := ag.Push("", nil)
	assert.Equal(t, constants.ErrBrokenPipe, err)
}

func TestAgentPushFailsIfChannelFull(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	// heartbeat and handshake if not set by another test
	// TODO: ugly ugly
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Not(gomock.Nil())).AnyTimes()
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Nil()).AnyTimes()

	ag := NewAgent(nil, nil, mockEncoder, nil, time.Second, 0, nil)
	assert.NotNil(t, ag)
	err := ag.Push("", nil)
	assert.Equal(t, constants.ErrBufferExceed, err)
}

func TestAgentPush(t *testing.T) {
	tables := []struct {
		name string
		data interface{}
		err  error
	}{
		{"success_raw", []byte("ok"), nil},
		{"success_struct", &someStruct{A: "ok"}, nil},
		{"failure", []byte("ok"), constants.ErrBrokenPipe},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
			// heartbeat and handshake if not set by another test
			// TODO: ugly ugly
			mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Not(gomock.Nil())).AnyTimes()
			mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Nil()).AnyTimes()
			mockDecoder := codecmocks.NewMockPacketDecoder(ctrl)
			dieChan := make(chan bool)
			hbTime := time.Second

			conn, _ := net.Dial("tcp", "127.0.0.1:0")
			ag := NewAgent(conn, mockDecoder, mockEncoder, mockSerializer, hbTime, 10, dieChan)
			assert.NotNil(t, ag)

			msg := pendingMessage{
				typ:     message.Push,
				route:   uuid.New().String(),
				payload: table.data,
			}

			if table.err != nil {
				close(ag.chSend)
			}
			err := ag.Push(msg.route, table.data)
			assert.Equal(t, table.err, err)

			if table.err == nil {
				recvMsg := helpers.ShouldEventuallyReceive(t, ag.chSend).(pendingMessage)
				assert.Equal(t, msg, recvMsg)
			}
		})
	}
}

func TestAgentResponseMIDFailsIfClosedAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	// heartbeat and handshake if not set by another test
	// TODO: ugly ugly
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Not(gomock.Nil())).AnyTimes()
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Nil()).AnyTimes()

	ag := NewAgent(nil, nil, mockEncoder, nil, time.Second, 10, nil)
	assert.NotNil(t, ag)
	ag.state = constants.StatusClosed
	err := ag.ResponseMID(1, nil)
	assert.Equal(t, constants.ErrBrokenPipe, err)
}

func TestAgentResponseMIDFailsIfChannelFull(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	// heartbeat and handshake if not set by another test
	// TODO: ugly ugly
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Not(gomock.Nil())).AnyTimes()
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Nil()).AnyTimes()

	ag := NewAgent(nil, nil, mockEncoder, nil, time.Second, 0, nil)
	assert.NotNil(t, ag)
	err := ag.ResponseMID(1, nil)
	assert.Equal(t, constants.ErrBufferExceed, err)
}

func TestAgentResponseMID(t *testing.T) {
	tables := []struct {
		name   string
		mid    uint
		data   interface{}
		msgErr bool
		err    error
	}{
		{"success_raw", uint(rand.Int()), []byte("ok"), false, nil},
		{"success_raw_msg_err", uint(rand.Int()), []byte("ok"), true, nil},
		{"success_struct", uint(rand.Int()), &someStruct{A: "ok"}, false, nil},
		{"failure_empty_mid", 0, []byte("ok"), false, constants.ErrSessionOnNotify},
		{"failure_send", uint(rand.Int()), []byte("ok"), false, constants.ErrBrokenPipe},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
			// heartbeat and handshake if not set by another test
			// TODO: ugly ugly
			mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Not(gomock.Nil())).AnyTimes()
			mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Nil()).AnyTimes()
			mockDecoder := codecmocks.NewMockPacketDecoder(ctrl)
			dieChan := make(chan bool)
			hbTime := time.Second

			conn, _ := net.Dial("tcp", "127.0.0.1:0")
			ag := NewAgent(conn, mockDecoder, mockEncoder, mockSerializer, hbTime, 10, dieChan)
			assert.NotNil(t, ag)

			var msg pendingMessage
			if table.mid != 0 {
				msg = pendingMessage{
					typ:     message.Response,
					mid:     table.mid,
					payload: table.data,
					err:     table.msgErr,
				}

				if table.err != nil {
					close(ag.chSend)
				}
			}
			var err error
			if table.msgErr {
				err = ag.ResponseMID(table.mid, table.data, table.msgErr)
			} else {
				err = ag.ResponseMID(table.mid, table.data)
			}
			assert.Equal(t, table.err, err)

			if table.err == nil {
				recvMsg := helpers.ShouldEventuallyReceive(t, ag.chSend).(pendingMessage)
				assert.Equal(t, msg, recvMsg)
			}
		})
	}
}
