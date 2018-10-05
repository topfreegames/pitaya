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
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/constants"
	pcontext "github.com/topfreegames/pitaya/context"
	e "github.com/topfreegames/pitaya/errors"
	"github.com/topfreegames/pitaya/helpers"
	codecmocks "github.com/topfreegames/pitaya/internal/codec/mocks"
	"github.com/topfreegames/pitaya/internal/message"
	messagemocks "github.com/topfreegames/pitaya/internal/message/mocks"
	"github.com/topfreegames/pitaya/internal/packet"
	"github.com/topfreegames/pitaya/metrics"
	metricsmocks "github.com/topfreegames/pitaya/metrics/mocks"
	"github.com/topfreegames/pitaya/mocks"
	"github.com/topfreegames/pitaya/protos"
	serializemocks "github.com/topfreegames/pitaya/serialize/mocks"
	"github.com/topfreegames/pitaya/session"
)

type mockAddr struct{}

func (m *mockAddr) Network() string { return "" }
func (m *mockAddr) String() string  { return "remote-string" }

func heartbeatAndHandshakeMocks(mockEncoder *codecmocks.MockPacketEncoder) {
	// heartbeat and handshake if not set by another test
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Not(gomock.Nil())).AnyTimes()
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Nil()).AnyTimes()
}

func getCtxWithRequestKeys() context.Context {
	ctx := pcontext.AddToPropagateCtx(context.Background(), constants.StartTimeKey, time.Now().UnixNano())
	return pcontext.AddToPropagateCtx(ctx, constants.RouteKey, "route")
}

func TestNewAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockSerializer.EXPECT().GetName()

	mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	mockDecoder := codecmocks.NewMockPacketDecoder(ctrl)
	dieChan := make(chan bool)
	hbTime := time.Second

	mockConn := mocks.NewMockConn(ctrl)

	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Not(gomock.Nil())).Do(
		func(typ packet.Type, d []byte) {
			// cannot compare inside the expect because they are equivalent but not equal
			assert.EqualValues(t, packet.Handshake, typ)
		})
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Nil()).Do(
		func(typ packet.Type, d []byte) {
			assert.EqualValues(t, packet.Heartbeat, typ)
		})
	messageEncoder := message.NewMessagesEncoder(false)

	mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}

	mockMetricsReporter.EXPECT().ReportGauge(metrics.ConnectedClients, gomock.Any(), gomock.Any())
	ag := NewAgent(mockConn, mockDecoder, mockEncoder, mockSerializer, hbTime, 10, dieChan, messageEncoder, mockMetricsReporters)
	assert.NotNil(t, ag)
	assert.IsType(t, make(chan struct{}), ag.chDie)
	assert.IsType(t, make(chan pendingMessage), ag.chSend)
	assert.IsType(t, make(chan struct{}), ag.chStopHeartbeat)
	assert.IsType(t, make(chan struct{}), ag.chStopWrite)
	assert.Equal(t, dieChan, ag.appDieChan)
	assert.Equal(t, 10, ag.messagesBufferSize)
	assert.Equal(t, mockConn, ag.conn)
	assert.Equal(t, mockDecoder, ag.decoder)
	assert.Equal(t, mockEncoder, ag.encoder)
	assert.Equal(t, hbTime, ag.heartbeatTimeout)
	assert.InDelta(t, time.Now().Unix(), ag.lastAt, 1)
	assert.Equal(t, mockSerializer, ag.serializer)
	assert.Equal(t, mockMetricsReporters, ag.metricsReporters)
	assert.Equal(t, constants.StatusStart, ag.state)
	assert.NotNil(t, ag.Session)
	assert.True(t, ag.Session.IsFrontend)

	// second call should no call hdb encode
	mockMetricsReporter.EXPECT().ReportGauge(metrics.ConnectedClients, gomock.Any(), gomock.Any())
	ag = NewAgent(nil, nil, mockEncoder, nil, hbTime, 10, dieChan, messageEncoder, mockMetricsReporters)
	assert.NotNil(t, ag)
}

func TestKick(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	mockDecoder := codecmocks.NewMockPacketDecoder(ctrl)
	dieChan := make(chan bool)
	hbTime := time.Second

	mockConn := mocks.NewMockConn(ctrl)
	mockEncoder.EXPECT().Encode(gomock.Any(), gomock.Nil()).Do(
		func(typ packet.Type, d []byte) {
			assert.EqualValues(t, packet.Kick, typ)
		})
	mockConn.EXPECT().Write(gomock.Any()).Return(0, nil)
	messageEncoder := message.NewMessagesEncoder(false)

	ag := NewAgent(mockConn, mockDecoder, mockEncoder, mockSerializer, hbTime, 10, dieChan, messageEncoder, nil)
	c := context.Background()
	err := ag.Kick(c)
	assert.NoError(t, err)
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
			heartbeatAndHandshakeMocks(mockEncoder)
			mockDecoder := codecmocks.NewMockPacketDecoder(ctrl)
			dieChan := make(chan bool)
			hbTime := time.Second
			messageEncoder := message.NewMessagesEncoder(false)

			mockConn := mocks.NewMockConn(ctrl)
			ag := NewAgent(mockConn, mockDecoder, mockEncoder, mockSerializer, hbTime, 10, dieChan, messageEncoder, nil)
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
	heartbeatAndHandshakeMocks(mockEncoder)
	messageEncoder := message.NewMessagesEncoder(false)

	ag := NewAgent(nil, nil, mockEncoder, nil, time.Second, 10, nil, messageEncoder, nil)
	assert.NotNil(t, ag)
	ag.state = constants.StatusClosed
	err := ag.Push("", nil)
	assert.Equal(t, constants.ErrBrokenPipe, err)
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
			heartbeatAndHandshakeMocks(mockEncoder)
			mockDecoder := codecmocks.NewMockPacketDecoder(ctrl)
			dieChan := make(chan bool)
			hbTime := time.Second
			messageEncoder := message.NewMessagesEncoder(false)
			mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)
			mockConn := mocks.NewMockConn(ctrl)
			mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}
			mockMetricsReporter.EXPECT().ReportGauge(metrics.ConnectedClients, gomock.Any(), gomock.Any())
			ag := NewAgent(mockConn, mockDecoder, mockEncoder, mockSerializer, hbTime, 10, dieChan, messageEncoder, mockMetricsReporters)
			assert.NotNil(t, ag)

			msg := pendingMessage{
				typ:     message.Push,
				route:   uuid.New().String(),
				payload: table.data,
			}

			if table.err != nil {
				close(ag.chSend)
			}

			mockMetricsReporter.EXPECT().ReportGauge(metrics.ChannelCapacity, gomock.Any(), float64(10))
			err := ag.Push(msg.route, table.data)
			assert.Equal(t, table.err, err)

			if table.err == nil {
				recvMsg := helpers.ShouldEventuallyReceive(t, ag.chSend).(pendingMessage)
				assert.Equal(t, msg, recvMsg)
			}
		})
	}
}

func TestAgentPushFullChannel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	heartbeatAndHandshakeMocks(mockEncoder)
	mockDecoder := codecmocks.NewMockPacketDecoder(ctrl)
	dieChan := make(chan bool)
	hbTime := time.Second
	messageEncoder := message.NewMessagesEncoder(false)
	mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)
	mockConn := mocks.NewMockConn(ctrl)
	mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}
	mockMetricsReporter.EXPECT().ReportGauge(metrics.ConnectedClients, gomock.Any(), gomock.Any())
	ag := NewAgent(mockConn, mockDecoder, mockEncoder, mockSerializer, hbTime, 0, dieChan, messageEncoder, mockMetricsReporters)
	assert.NotNil(t, ag)

	mockMetricsReporter.EXPECT().ReportGauge(metrics.ChannelCapacity, gomock.Any(), float64(0))
	go func() {
		err := ag.Push("route", []byte("data"))
		assert.NoError(t, err)
	}()
	helpers.ShouldEventuallyReceive(t, ag.chSend)
}

func TestAgentResponseMIDFailsIfClosedAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	heartbeatAndHandshakeMocks(mockEncoder)
	mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)
	mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)

	mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}
	mockMetricsReporter.EXPECT().ReportGauge(metrics.ConnectedClients, gomock.Any(), gomock.Any())
	ag := NewAgent(nil, nil, mockEncoder, nil, time.Second, 10, nil, mockMessageEncoder, mockMetricsReporters)
	assert.NotNil(t, ag)
	ag.state = constants.StatusClosed

	mockMetricsReporters[0].(*metricsmocks.MockReporter).EXPECT().ReportSummary(metrics.ResponseTime, gomock.Any(), gomock.Any())
	ctx := getCtxWithRequestKeys()
	err := ag.ResponseMID(ctx, 1, nil)
	assert.Equal(t, constants.ErrBrokenPipe, err)
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

			mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)
			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
			heartbeatAndHandshakeMocks(mockEncoder)
			mockDecoder := codecmocks.NewMockPacketDecoder(ctrl)
			dieChan := make(chan bool)
			hbTime := time.Second
			messageEncoder := message.NewMessagesEncoder(false)

			mockConn := mocks.NewMockConn(ctrl)
			mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}
			mockMetricsReporter.EXPECT().ReportGauge(metrics.ConnectedClients, gomock.Any(), gomock.Any())
			ag := NewAgent(mockConn, mockDecoder, mockEncoder, mockSerializer, hbTime, 10, dieChan, messageEncoder, mockMetricsReporters)
			assert.NotNil(t, ag)

			ctx := getCtxWithRequestKeys()
			if table.mid == 0 {
				mockMetricsReporters[0].(*metricsmocks.MockReporter).EXPECT().ReportSummary(metrics.ResponseTime, gomock.Any(), gomock.Any())
			} else {
				mockMetricsReporter.EXPECT().ReportGauge(metrics.ChannelCapacity, gomock.Any(), float64(10))
			}
			var msg pendingMessage
			if table.mid != 0 {
				msg = pendingMessage{
					ctx:     ctx,
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
				err = ag.ResponseMID(ctx, table.mid, table.data, table.msgErr)
			} else {
				err = ag.ResponseMID(ctx, table.mid, table.data)
			}
			assert.Equal(t, table.err, err)

			if table.err == nil {
				recvMsg := helpers.ShouldEventuallyReceive(t, ag.chSend).(pendingMessage)
				assert.Equal(t, msg, recvMsg)
			}
		})
	}
}

func TestAgentResponseMIDFullChannel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	heartbeatAndHandshakeMocks(mockEncoder)
	mockDecoder := codecmocks.NewMockPacketDecoder(ctrl)
	dieChan := make(chan bool)
	hbTime := time.Second
	messageEncoder := message.NewMessagesEncoder(false)
	mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)
	mockConn := mocks.NewMockConn(ctrl)
	mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}
	mockMetricsReporter.EXPECT().ReportGauge(metrics.ConnectedClients, gomock.Any(), gomock.Any())
	ag := NewAgent(mockConn, mockDecoder, mockEncoder, mockSerializer, hbTime, 0, dieChan, messageEncoder, mockMetricsReporters)
	assert.NotNil(t, ag)
	mockMetricsReporters[0].(*metricsmocks.MockReporter).EXPECT().ReportGauge(metrics.ChannelCapacity, gomock.Any(), float64(0))
	go func() {
		err := ag.ResponseMID(nil, 1, []byte("data"))
		assert.NoError(t, err)
	}()
	helpers.ShouldEventuallyReceive(t, ag.chSend)
}

func TestAgentCloseFailsIfAlreadyClosed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	heartbeatAndHandshakeMocks(mockEncoder)
	mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)

	ag := NewAgent(nil, nil, mockEncoder, nil, time.Second, 10, nil, mockMessageEncoder, nil)
	assert.NotNil(t, ag)
	ag.state = constants.StatusClosed
	err := ag.Close()
	assert.Equal(t, constants.ErrCloseClosedSession, err)
}

func TestAgentClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mocks.NewMockConn(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	heartbeatAndHandshakeMocks(mockEncoder)
	mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)
	ag := NewAgent(mockConn, nil, mockEncoder, nil, time.Second, 0, nil, mockMessageEncoder, nil)
	assert.NotNil(t, ag)

	expected := false
	f := func() { expected = true }
	err := ag.Session.OnClose(f)
	assert.NoError(t, err)

	// validate channels are closed
	stopWrite := false
	stopHeartbeat := false
	die := false
	go func() {
		for {
			select {
			case <-ag.chStopWrite:
				stopWrite = true
			case <-ag.chStopHeartbeat:
				stopHeartbeat = true
			case <-ag.chDie:
				die = true
			}
		}
	}()

	mockConn.EXPECT().RemoteAddr()
	mockConn.EXPECT().Close()
	err = ag.Close()
	assert.NoError(t, err)
	assert.Equal(t, ag.state, constants.StatusClosed)
	assert.True(t, expected)
	helpers.ShouldEventuallyReturn(
		t, func() bool { return stopWrite && stopHeartbeat && die },
		true, 50*time.Millisecond, 500*time.Millisecond)
}

func TestAgentRemoteAddr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mocks.NewMockConn(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	heartbeatAndHandshakeMocks(mockEncoder)
	mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)
	ag := NewAgent(mockConn, nil, mockEncoder, nil, time.Second, 0, nil, mockMessageEncoder, nil)
	assert.NotNil(t, ag)

	expected := &mockAddr{}
	mockConn.EXPECT().RemoteAddr().Return(expected)
	addr := ag.RemoteAddr()
	assert.Equal(t, expected, addr)
}

func TestAgentString(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mocks.NewMockConn(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	heartbeatAndHandshakeMocks(mockEncoder)
	mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)
	ag := NewAgent(mockConn, nil, mockEncoder, nil, time.Second, 0, nil, mockMessageEncoder, nil)
	assert.NotNil(t, ag)

	mockConn.EXPECT().RemoteAddr().Return(&mockAddr{})
	expected := fmt.Sprintf("Remote=remote-string, LastTime=%d", ag.lastAt)
	str := ag.String()
	assert.Equal(t, expected, str)
}

func TestAgentGetStatus(t *testing.T) {
	tables := []struct {
		name   string
		status int32
	}{
		{"start", constants.StatusStart},
		{"closed", constants.StatusClosed},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConn := mocks.NewMockConn(ctrl)
			mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
			heartbeatAndHandshakeMocks(mockEncoder)
			mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)
			ag := NewAgent(mockConn, nil, mockEncoder, nil, time.Second, 0, nil, mockMessageEncoder, nil)
			assert.NotNil(t, ag)

			ag.state = table.status

			status := ag.GetStatus()
			assert.Equal(t, table.status, status)
		})
	}
}

func TestAgentSetLastAt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	heartbeatAndHandshakeMocks(mockEncoder)
	mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)
	ag := NewAgent(nil, nil, mockEncoder, nil, time.Second, 0, nil, mockMessageEncoder, nil)
	assert.NotNil(t, ag)

	ag.lastAt = 0
	ag.SetLastAt()
	assert.InDelta(t, time.Now().Unix(), ag.lastAt, 1)
}

func TestAgentSetStatus(t *testing.T) {
	tables := []struct {
		name   string
		status int32
	}{
		{"start", constants.StatusStart},
		{"closed", constants.StatusClosed},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
			heartbeatAndHandshakeMocks(mockEncoder)
			mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)
			ag := NewAgent(nil, nil, mockEncoder, nil, time.Second, 0, nil, mockMessageEncoder, nil)
			assert.NotNil(t, ag)

			ag.SetStatus(table.status)
			assert.Equal(t, table.status, ag.state)
		})
	}
}

func TestOnSessionClosed(t *testing.T) {
	ss := session.New(nil, true)

	expected := false
	f := func() { expected = true }
	err := ss.OnClose(f)
	assert.NoError(t, err)

	assert.NotPanics(t, func() { onSessionClosed(ss) })
	assert.True(t, expected)
}

func TestOnSessionClosedRecoversIfPanic(t *testing.T) {
	ss := session.New(nil, true)

	expected := false
	f := func() {
		expected = true
		panic("oh noes")
	}
	err := ss.OnClose(f)
	assert.NoError(t, err)

	assert.NotPanics(t, func() { onSessionClosed(ss) })
	assert.True(t, expected)
}

func TestAgentSendHandshakeResponse(t *testing.T) {
	tables := []struct {
		name string
		err  error
	}{
		{"success", nil},
		{"failure", errors.New("handshake failed")},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConn := mocks.NewMockConn(ctrl)
			mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
			heartbeatAndHandshakeMocks(mockEncoder)
			mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)
			ag := NewAgent(mockConn, nil, mockEncoder, nil, time.Second, 0, nil, mockMessageEncoder, nil)
			assert.NotNil(t, ag)

			mockConn.EXPECT().Write(hrd).Return(0, table.err)
			err := ag.SendHandshakeResponse()
			assert.Equal(t, table.err, err)
		})
	}
}

func TestAnswerWithError(t *testing.T) {
	tables := []struct {
		name          string
		getPayloadErr error
		resErr        error
		err           error
	}{
		{"success", nil, nil, nil},
		{"failure_get_payload", errors.New("serialize err"), nil, errors.New("serialize err")},
		{"failure_response_mid", nil, errors.New("responsemid err"), errors.New("responsemid err")},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSerializer := serializemocks.NewMockSerializer(ctrl)
			mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
			heartbeatAndHandshakeMocks(mockEncoder)
			messageEncoder := message.NewMessagesEncoder(false)
			ag := NewAgent(nil, nil, mockEncoder, mockSerializer, time.Second, 1, nil, messageEncoder, nil)
			assert.NotNil(t, ag)

			mockSerializer.EXPECT().Marshal(gomock.Any()).Return(nil, table.getPayloadErr)
			ag.AnswerWithError(nil, uint(rand.Int()), errors.New("something went wrong"))
			if table.err == nil {
				helpers.ShouldEventuallyReceive(t, ag.chSend)
			}
		})
	}
}

func TestAgentHeartbeat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	heartbeatAndHandshakeMocks(mockEncoder)
	mockConn := mocks.NewMockConn(ctrl)
	mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)
	ag := NewAgent(mockConn, nil, mockEncoder, mockSerializer, 1*time.Second, 1, nil, mockMessageEncoder, nil)
	assert.NotNil(t, ag)

	mockConn.EXPECT().RemoteAddr().MaxTimes(1)
	mockConn.EXPECT().Close().MaxTimes(1)

	// Sends two heartbeats and then times out
	mockConn.EXPECT().Write(hbd).Return(0, nil).Times(2)
	die := false
	go func() {
		for {
			select {
			case <-ag.chDie:
				die = true
			}
		}
	}()

	go ag.heartbeat()
	helpers.ShouldEventuallyReturn(t, func() bool { return die }, true, 500*time.Millisecond, 5*time.Second)
}

func TestAgentHeartbeatExitsIfConnError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	heartbeatAndHandshakeMocks(mockEncoder)
	mockConn := mocks.NewMockConn(ctrl)
	mockMessageEncoder := messagemocks.NewMockEncoder(ctrl)
	ag := NewAgent(mockConn, nil, mockEncoder, mockSerializer, 1*time.Second, 1, nil, mockMessageEncoder, nil)
	assert.NotNil(t, ag)

	mockConn.EXPECT().RemoteAddr().MaxTimes(1)
	mockConn.EXPECT().Close().MaxTimes(1)

	mockConn.EXPECT().Write(hbd).Return(0, errors.New("broken"))
	die := false
	go func() {
		for {
			select {
			case <-ag.chDie:
				die = true
			}
		}
	}()

	go ag.heartbeat()
	helpers.ShouldEventuallyReturn(t, func() bool { return die }, true, 500*time.Millisecond, 2*time.Second)
}

func TestAgentHeartbeatExitsOnStopHeartbeat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	heartbeatAndHandshakeMocks(mockEncoder)
	mockConn := mocks.NewMockConn(ctrl)
	messageEncoder := message.NewMessagesEncoder(false)

	mockConn.EXPECT().RemoteAddr().MaxTimes(1)
	mockConn.EXPECT().Close().MaxTimes(1)

	ag := NewAgent(mockConn, nil, mockEncoder, mockSerializer, 1*time.Second, 1, nil, messageEncoder, nil)
	assert.NotNil(t, ag)

	go func() {
		time.Sleep(500 * time.Millisecond)
		ag.Close()
	}()

	ag.heartbeat()
}

func TestAgentWriteChSend(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	mockConn := mocks.NewMockConn(ctrl)
	messageEncoder := message.NewMessagesEncoder(false)
	mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)
	mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}
	ag := &Agent{ // avoid heartbeat and handshake to fully test serialize
		conn:             mockConn,
		chSend:           make(chan pendingMessage, 1),
		encoder:          mockEncoder,
		heartbeatTimeout: time.Second,
		lastAt:           time.Now().Unix(),
		serializer:       mockSerializer,
		messageEncoder:   messageEncoder,
		metricsReporters: mockMetricsReporters,
	}
	ctx := getCtxWithRequestKeys()
	mockMetricsReporters[0].(*metricsmocks.MockReporter).EXPECT().ReportSummary(metrics.ResponseTime, gomock.Any(), gomock.Any())

	expected := pendingMessage{
		ctx:     ctx,
		typ:     message.Response,
		route:   uuid.New().String(),
		mid:     uint(rand.Int()),
		payload: someStruct{A: "bla"},
	}
	expectedBytes := []byte("bla")
	mockSerializer.EXPECT().Marshal(expected.payload).Return(expectedBytes, nil)
	m := &message.Message{
		Type:  expected.typ,
		Data:  expectedBytes,
		Route: expected.route,
		ID:    expected.mid,
		Err:   false,
	}

	em, err := messageEncoder.Encode(m)
	assert.NoError(t, err)
	expectedPacket := []byte("final")
	mockEncoder.EXPECT().Encode(gomock.Any(), em).Return(expectedPacket, nil)
	var wg sync.WaitGroup
	wg.Add(1)
	mockConn.EXPECT().Write(expectedPacket).Do(func(b []byte) {
		wg.Done()
	})
	go ag.write()
	ag.chSend <- expected
	wg.Wait()
}

func TestAgentWriteChSendSerializeErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mocks.NewMockConn(ctrl)
	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	messageEncoder := message.NewMessagesEncoder(false)
	mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)
	mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}
	ag := &Agent{ // avoid heartbeat and handshake to fully test serialize
		conn:             mockConn,
		chSend:           make(chan pendingMessage, 1),
		encoder:          mockEncoder,
		heartbeatTimeout: time.Second,
		lastAt:           time.Now().Unix(),
		serializer:       mockSerializer,
		messageEncoder:   messageEncoder,
		metricsReporters: mockMetricsReporters,
		Session:          session.New(nil, true),
	}

	ctx := getCtxWithRequestKeys()
	mockMetricsReporters[0].(*metricsmocks.MockReporter).EXPECT().ReportSummary(metrics.ResponseTime, gomock.Any(), gomock.Any())

	expected := pendingMessage{
		ctx:     ctx,
		typ:     message.Response,
		route:   uuid.New().String(),
		mid:     uint(rand.Int()),
		payload: someStruct{A: "bla"},
	}
	expectedErr := errors.New("noo")
	mockSerializer.EXPECT().Marshal(expected.payload).Return(nil, expectedErr)

	expectedBytes := []byte("bla")
	mockSerializer.EXPECT().Marshal(&protos.Error{
		Code: e.ErrUnknownCode,
		Msg:  expectedErr.Error(),
	}).Return(expectedBytes, nil)
	m := &message.Message{
		Type:  expected.typ,
		Data:  expectedBytes,
		Route: expected.route,
		ID:    expected.mid,
		Err:   false,
	}
	em, err := messageEncoder.Encode(m)
	assert.NoError(t, err)
	expectedPacket := []byte("final")
	mockEncoder.EXPECT().Encode(gomock.Any(), em).Return(expectedPacket, nil)

	var wg sync.WaitGroup
	wg.Add(1)
	mockConn.EXPECT().Write(expectedPacket).Do(func(b []byte) {
		wg.Done()
	})
	go ag.write()
	ag.chSend <- expected
	wg.Wait()

}

func TestAgentHandle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	heartbeatAndHandshakeMocks(mockEncoder)
	mockConn := mocks.NewMockConn(ctrl)
	messageEncoder := message.NewMessagesEncoder(false)
	ag := NewAgent(mockConn, nil, mockEncoder, mockSerializer, 1*time.Second, 1, nil, messageEncoder, nil)
	assert.NotNil(t, ag)

	go ag.Handle()
	expected := pendingMessage{
		typ:     message.Request,
		route:   uuid.New().String(),
		mid:     uint(rand.Int()),
		payload: someStruct{A: "bla"},
	}

	mockSerializer.EXPECT().Marshal(expected.payload).Return(nil, nil)
	// Sends two heartbeats and then times out
	mockConn.EXPECT().Write(hbd).Return(0, nil).Times(2)
	var wg sync.WaitGroup
	wg.Add(1)
	closed := false
	go func() {
		for {
			select {
			case <-ag.chDie:
				closed = true
			}
		}
	}()

	mockConn.EXPECT().Write(gomock.Any()).Return(0, nil).Do(func(d []byte) {
		wg.Done()
	})

	// ag.Close on method exit
	mockConn.EXPECT().RemoteAddr().MaxTimes(1)
	mockConn.EXPECT().Close().MaxTimes(1)

	ag.chSend <- expected

	wg.Wait()
	helpers.ShouldEventuallyReturn(t, func() bool { return closed }, true, 50*time.Millisecond, 5*time.Second)
}

func TestNatsRPCServerReportMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializer := serializemocks.NewMockSerializer(ctrl)
	mockEncoder := codecmocks.NewMockPacketEncoder(ctrl)
	heartbeatAndHandshakeMocks(mockEncoder)
	mockDecoder := codecmocks.NewMockPacketDecoder(ctrl)
	dieChan := make(chan bool)
	hbTime := time.Second
	messageEncoder := message.NewMessagesEncoder(false)
	mockMetricsReporter := metricsmocks.NewMockReporter(ctrl)
	mockMetricsReporters := []metrics.Reporter{mockMetricsReporter}
	mockConn := mocks.NewMockConn(ctrl)
	mockMetricsReporter.EXPECT().ReportGauge(metrics.ConnectedClients, gomock.Any(), gomock.Any())
	ag := NewAgent(mockConn, mockDecoder, mockEncoder, mockSerializer, hbTime, 10, dieChan, messageEncoder, mockMetricsReporters)
	assert.NotNil(t, ag)

	ag.messagesBufferSize = 0

	ag.chSend <- pendingMessage{}

	mockMetricsReporter.EXPECT().ReportGauge(metrics.ChannelCapacity, gomock.Any(), float64(-1)) // because buffersize is 0 and chan sz is 1
	ag.reportChannelSize()
}

type customMockAddr struct{ network, str string }

func (m *customMockAddr) Network() string { return m.network }
func (m *customMockAddr) String() string  { return m.str }

func TestIPVersion(t *testing.T) {
	tables := []struct {
		addr      string
		ipVersion string
	}{
		{"127.0.0.1:80", constants.IPv4},
		{"1.2.3.4:3333", constants.IPv4},
		{"::1:3333", constants.IPv6},
		{"2001:db8:0000:1:1:1:1:1:3333", constants.IPv6},
	}

	for _, table := range tables {
		t.Run("test_"+table.addr, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockConn := mocks.NewMockConn(ctrl)
			mockAddr := &customMockAddr{str: table.addr}

			mockConn.EXPECT().RemoteAddr().Return(mockAddr)
			a := &Agent{conn: mockConn}

			assert.Equal(t, table.ipVersion, a.IPVersion())
		})
	}
}
