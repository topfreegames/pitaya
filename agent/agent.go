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
	gojson "encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/internal/packet"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/metrics"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/serialize"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/tracing"
	"github.com/topfreegames/pitaya/util"
	"github.com/topfreegames/pitaya/util/compression"
)

var (
	// hbd contains the heartbeat packet data
	hbd []byte
	// hrd contains the handshake response data
	hrd  []byte
	once sync.Once
)

const handlerType = "handler"

type (
	// Agent corresponds to a user and is used for storing raw Conn information
	Agent struct {
		Session            *session.Session    // session
		appDieChan         chan bool           // app die channel
		chDie              chan struct{}       // wait for close
		chSend             chan pendingMessage // push message queue
		chStopHeartbeat    chan struct{}       // stop heartbeats
		chStopWrite        chan struct{}       // stop writing messages
		conn               net.Conn            // low-level conn fd
		decoder            codec.PacketDecoder // binary decoder
		encoder            codec.PacketEncoder // binary encoder
		heartbeatTimeout   time.Duration
		lastAt             int64                // last heartbeat unix time stamp
		messagesBufferSize int                  // size of the pending messages buffer
		serializer         serialize.Serializer // message serializer
		state              int32                // current agent state
		messageEncoder     message.Encoder
		metricsReporters   []metrics.Reporter
	}

	pendingMessage struct {
		ctx     context.Context
		typ     message.Type // message type
		route   string       // message route (push)
		mid     uint         // response message id (response)
		payload interface{}  // payload
		err     bool         // if its an error message
	}
)

// NewAgent create new agent instance
func NewAgent(
	conn net.Conn,
	packetDecoder codec.PacketDecoder,
	packetEncoder codec.PacketEncoder,
	serializer serialize.Serializer,
	heartbeatTime time.Duration,
	messagesBufferSize int,
	dieChan chan bool,
	messageEncoder message.Encoder,
	metricsReporters []metrics.Reporter,
) *Agent {
	// initialize heartbeat and handshake data on first player connection
	once.Do(func() {
		hbdEncode(heartbeatTime, packetEncoder, messageEncoder.IsCompressionEnabled(), serializer.GetName())
	})

	a := &Agent{
		appDieChan:         dieChan,
		chDie:              make(chan struct{}),
		chSend:             make(chan pendingMessage, messagesBufferSize),
		messagesBufferSize: messagesBufferSize,
		chStopHeartbeat:    make(chan struct{}),
		chStopWrite:        make(chan struct{}),
		conn:               conn,
		decoder:            packetDecoder,
		encoder:            packetEncoder,
		heartbeatTimeout:   heartbeatTime,
		lastAt:             time.Now().Unix(),
		serializer:         serializer,
		state:              constants.StatusStart,
		messageEncoder:     messageEncoder,
		metricsReporters:   metricsReporters,
	}

	// binding session
	s := session.New(a, true)
	metrics.ReportNumberOfConnectedClients(metricsReporters, session.SessionCount)
	a.Session = s
	return a
}

func (a *Agent) send(m pendingMessage) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = constants.ErrBrokenPipe
		}
	}()
	a.reportChannelSize()
	a.chSend <- m
	return
}

// Push implementation for session.NetworkEntity interface
func (a *Agent) Push(route string, v interface{}) error {
	if a.GetStatus() == constants.StatusClosed {
		return constants.ErrBrokenPipe
	}

	switch d := v.(type) {
	case []byte:
		logger.Log.Debugf("Type=Push, ID=%d, UID=%d, Route=%s, Data=%dbytes",
			a.Session.ID(), a.Session.UID(), route, len(d))
	default:
		logger.Log.Debugf("Type=Push, ID=%d, UID=%d, Route=%s, Data=%+v",
			a.Session.ID(), a.Session.UID(), route, v)
	}
	return a.send(pendingMessage{typ: message.Push, route: route, payload: v})
}

// ResponseMID implementation for session.NetworkEntity interface
// Response message to session
func (a *Agent) ResponseMID(ctx context.Context, mid uint, v interface{}, isError ...bool) error {
	err := false
	if len(isError) > 0 {
		err = isError[0]
	}
	if a.GetStatus() == constants.StatusClosed {
		err := constants.ErrBrokenPipe
		tracing.FinishSpan(ctx, err)
		metrics.ReportTimingFromCtx(ctx, a.metricsReporters, handlerType, true)
		return err
	}

	if mid <= 0 {
		err := constants.ErrSessionOnNotify
		tracing.FinishSpan(ctx, err)
		metrics.ReportTimingFromCtx(ctx, a.metricsReporters, handlerType, true)
		return err
	}

	switch d := v.(type) {
	case []byte:
		logger.Log.Debugf("Type=Response, ID=%d, UID=%d, MID=%d, Data=%dbytes",
			a.Session.ID(), a.Session.UID(), mid, len(d))
	default:
		logger.Log.Infof("Type=Response, ID=%d, UID=%d, MID=%d, Data=%+v",
			a.Session.ID(), a.Session.UID(), mid, v)
	}

	return a.send(pendingMessage{ctx: ctx, typ: message.Response, mid: mid, payload: v, err: err})
}

// Close closes the agent, cleans inner state and closes low-level connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (a *Agent) Close() error {
	if a.GetStatus() == constants.StatusClosed {
		return constants.ErrCloseClosedSession
	}
	a.SetStatus(constants.StatusClosed)

	logger.Log.Debugf("Session closed, ID=%d, UID=%d, IP=%s",
		a.Session.ID(), a.Session.UID(), a.conn.RemoteAddr())

	// prevent closing closed channel
	select {
	case <-a.chDie:
		// expect
	default:
		close(a.chStopWrite)
		close(a.chStopHeartbeat)
		close(a.chDie)
		onSessionClosed(a.Session)
	}

	metrics.ReportNumberOfConnectedClients(a.metricsReporters, session.SessionCount)

	return a.conn.Close()
}

// RemoteAddr implementation for session.NetworkEntity interface
// returns the remote network address.
func (a *Agent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

// String, implementation for Stringer interface
func (a *Agent) String() string {
	return fmt.Sprintf("Remote=%s, LastTime=%d", a.conn.RemoteAddr().String(), a.lastAt)
}

// GetStatus gets the status
func (a *Agent) GetStatus() int32 {
	return atomic.LoadInt32(&a.state)
}

// Kick sends a kick packet to a client
func (a *Agent) Kick(ctx context.Context) error {
	// packet encode
	p, err := a.encoder.Encode(packet.Kick, nil)
	if err != nil {
		return err
	}
	_, err = a.conn.Write(p)
	return err
}

// SetLastAt sets the last at to now
func (a *Agent) SetLastAt() {
	a.lastAt = time.Now().Unix()
}

// SetStatus sets the agent status
func (a *Agent) SetStatus(state int32) {
	atomic.StoreInt32(&a.state, state)
}

// Handle handles the messages from and to a client
func (a *Agent) Handle() {
	defer func() {
		a.Close()
		logger.Log.Debugf("Session handle goroutine exit, SessionID=%d, UID=%d", a.Session.ID(), a.Session.UID())
	}()

	go a.write()
	go a.heartbeat()
	select {
	case <-a.chDie: // agent closed signal
		return
	}
}

func (a *Agent) heartbeat() {
	ticker := time.NewTicker(a.heartbeatTimeout)

	defer func() {
		ticker.Stop()
		a.Close()
	}()

	for {
		select {
		case <-ticker.C:
			deadline := time.Now().Add(-2 * a.heartbeatTimeout).Unix()
			if a.lastAt < deadline {
				logger.Log.Debugf("Session heartbeat timeout, LastTime=%d, Deadline=%d", a.lastAt, deadline)
				return
			}
			if _, err := a.conn.Write(hbd); err != nil {
				return
			}
		case <-a.chDie:
			return
		case <-a.chStopHeartbeat:
			return
		}
	}
}

func onSessionClosed(s *session.Session) {
	defer func() {
		if err := recover(); err != nil {
			logger.Log.Errorf("pitaya/onSessionClosed: %v", err)
		}
	}()

	for _, fn1 := range s.OnCloseCallbacks {
		fn1()
	}

	for _, fn2 := range session.SessionCloseCallbacks {
		fn2(s)
	}
}

// SendHandshakeResponse sends a handshake response
func (a *Agent) SendHandshakeResponse() error {
	_, err := a.conn.Write(hrd)
	return err
}

func (a *Agent) write() {
	// clean func
	defer func() {
		close(a.chSend)
		a.Close()
	}()

	for {
		select {
		case data := <-a.chSend:
			payload, err := util.SerializeOrRaw(a.serializer, data.payload)
			if err != nil {
				logger.Log.Error(err.Error())
				payload, err = util.GetErrorPayload(a.serializer, err)
				if err != nil {
					tracing.FinishSpan(data.ctx, err)
					if data.typ == message.Response {
						metrics.ReportTimingFromCtx(data.ctx, a.metricsReporters, handlerType, true)
					}
					logger.Log.Error("cannot serialize message and respond to the client ", err.Error())
					break
				}
			}

			// construct message and encode
			m := &message.Message{
				Type:  data.typ,
				Data:  payload,
				Route: data.route,
				ID:    data.mid,
				Err:   data.err,
			}
			em, err := a.messageEncoder.Encode(m)
			if err != nil {
				tracing.FinishSpan(data.ctx, err)
				if data.typ == message.Response {
					metrics.ReportTimingFromCtx(data.ctx, a.metricsReporters, handlerType, true)
				}
				logger.Log.Error(err.Error())
				break
			}

			// packet encode
			p, err := a.encoder.Encode(packet.Data, em)
			if err != nil {
				tracing.FinishSpan(data.ctx, err)
				if data.typ == message.Response {
					metrics.ReportTimingFromCtx(data.ctx, a.metricsReporters, handlerType, true)
				}
				logger.Log.Error(err)
				break
			}
			// close agent if low-level Conn broken
			if _, err := a.conn.Write(p); err != nil {
				tracing.FinishSpan(data.ctx, err)
				if data.typ == message.Response {
					metrics.ReportTimingFromCtx(data.ctx, a.metricsReporters, handlerType, true)
				}
				logger.Log.Error(err.Error())
				return
			}
			var e error
			tracing.FinishSpan(data.ctx, e)
			if data.typ == message.Response {
				metrics.ReportTimingFromCtx(data.ctx, a.metricsReporters, handlerType, m.Err)
			}
		case <-a.chStopWrite:
			return
		}
	}
}

// SendRequest sends a request to a server
func (a *Agent) SendRequest(ctx context.Context, serverID, route string, v interface{}) (*protos.Response, error) {
	return nil, errors.New("not implemented")
}

// AnswerWithError answers with an error
func (a *Agent) AnswerWithError(ctx context.Context, mid uint, err error) {
	if ctx != nil && err != nil {
		s := opentracing.SpanFromContext(ctx)
		if s != nil {
			tracing.LogError(s, err.Error())
		}
	}
	p, e := util.GetErrorPayload(a.serializer, err)
	if e != nil {
		logger.Log.Error("error answering the player with an error: ", e.Error())
		return
	}
	e = a.Session.ResponseMID(ctx, mid, p, true)
	if e != nil {
		logger.Log.Error("error answering the player with an error: ", e.Error())
	}
}

func hbdEncode(heartbeatTimeout time.Duration, packetEncoder codec.PacketEncoder, dataCompression bool, serializerName string) {
	hData := map[string]interface{}{
		"code": 200,
		"sys": map[string]interface{}{
			"heartbeat":  heartbeatTimeout.Seconds(),
			"dict":       message.GetDictionary(),
			"serializer": serializerName,
		},
	}
	data, err := gojson.Marshal(hData)
	if err != nil {
		panic(err)
	}

	if dataCompression {
		compressedData, err := compression.DeflateData(data)
		if err != nil {
			panic(err)
		}

		if len(compressedData) < len(data) {
			data = compressedData
		}
	}

	hrd, err = packetEncoder.Encode(packet.Handshake, data)
	if err != nil {
		panic(err)
	}

	hbd, err = packetEncoder.Encode(packet.Heartbeat, nil)
	if err != nil {
		panic(err)
	}
}

func (a *Agent) reportChannelSize() {
	chSendCapacity := a.messagesBufferSize - len(a.chSend)
	if chSendCapacity == 0 {
		logger.Log.Warnf("chSend is at maximum capacity")
	}
	for _, mr := range a.metricsReporters {
		if err := mr.ReportGauge(metrics.ChannelCapacity, map[string]string{"channel": "agent_chsend"}, float64(chSendCapacity)); err != nil {
			logger.Log.Warnf("failed to report chSend channel capaacity: %s", err.Error())
		}
	}
}
