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
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lonnng/nano/serialize"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/internal/packet"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/pipeline"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/util"
)

const (
	agentWriteBacklog = 16
)

var (
	log = logger.Log
	hbd []byte // heartbeat packet data
	// Hrd contains the handshake response data
	Hrd  []byte // handshake response data
	once sync.Once
)

type (
	// Agent corresponding a user, used for store raw Conn information
	Agent struct {
		// regular agent member
		Session          *session.Session               // session
		Conn             net.Conn                       // low-level conn fd
		lastMid          uint                           // last message id
		state            int32                          // current agent state
		chDie            chan struct{}                  // wait for close
		chSend           chan pendingMessage            // push message queue
		chRecv           chan *message.UnhandledMessage // unhandledMessages
		chWrite          chan []byte                    // write message to the clients
		chStopWrite      chan struct{}                  // stop writing messages
		chStopHeartbeat  chan struct{}                  // stop heartbeats
		chStopRead       chan struct{}                  //stop reading
		lastAt           int64                          // last heartbeat unix time stamp
		decoder          codec.PacketDecoder            // binary decoder
		encoder          codec.PacketEncoder            // binary encoder
		serializer       serialize.Serializer           // message serializer
		appDieChan       chan bool                      // app die channel
		heartbeatTimeout time.Duration

		Srv reflect.Value // cached session reflect.Value, this avoids repeated calls to reflect.value(a.Session)
	}

	pendingMessage struct {
		typ     message.Type // message type
		route   string       // message route(push)
		mid     uint         // response message id(response)
		payload interface{}  // payload
	}
)

// NewAgent create new agent instance
func NewAgent(
	conn net.Conn,
	packetDecoder codec.PacketDecoder,
	packetEncoder codec.PacketEncoder,
	serializer serialize.Serializer,
	heartbeatTime time.Duration,
	dieChan chan bool,
) *Agent {
	// initialize heartbeat and handshake data on first player connection
	once.Do(func() {
		hbdEncode(heartbeatTime, packetEncoder)
	})

	a := &Agent{
		Conn:             conn,
		state:            constants.StatusStart,
		chDie:            make(chan struct{}),
		chStopWrite:      make(chan struct{}),
		chStopHeartbeat:  make(chan struct{}),
		chStopRead:       make(chan struct{}),
		chWrite:          make(chan []byte, agentWriteBacklog),
		lastAt:           time.Now().Unix(),
		chSend:           make(chan pendingMessage, agentWriteBacklog),
		chRecv:           make(chan *message.UnhandledMessage),
		decoder:          packetDecoder,
		encoder:          packetEncoder,
		serializer:       serializer,
		heartbeatTimeout: heartbeatTime,
		appDieChan:       dieChan,
	}

	// binding session
	s := session.New(a)
	a.Session = s
	a.Srv = reflect.ValueOf(s)

	return a
}

func (a *Agent) send(m pendingMessage) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = constants.ErrBrokenPipe
		}
	}()
	a.chSend <- m
	return
}

// MID gets the last message id
func (a *Agent) MID() uint {
	return a.lastMid
}

// Push implementation for session.NetworkEntity interface
func (a *Agent) Push(route string, v interface{}) error {
	if a.GetStatus() == constants.StatusClosed {
		return constants.ErrBrokenPipe
	}

	if len(a.chSend) >= agentWriteBacklog {
		return constants.ErrBufferExceed
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

// Response implementation for session.NetworkEntity interface
// Response message to session
func (a *Agent) Response(v interface{}) error {
	return a.ResponseMID(a.lastMid, v)
}

// ResponseMID implementation for session.NetworkEntity interface
// Response message to session
func (a *Agent) ResponseMID(mid uint, v interface{}) error {
	if a.GetStatus() == constants.StatusClosed {
		return constants.ErrBrokenPipe
	}

	if mid <= 0 {
		return constants.ErrSessionOnNotify
	}

	if len(a.chSend) >= agentWriteBacklog {
		return constants.ErrBufferExceed
	}

	switch d := v.(type) {
	case []byte:
		logger.Log.Debugf("Type=Response, ID=%d, UID=%d, MID=%d, Data=%dbytes",
			a.Session.ID(), a.Session.UID(), mid, len(d))
	default:
		logger.Log.Infof("Type=Response, ID=%d, UID=%d, MID=%d, Data=%+v",
			a.Session.ID(), a.Session.UID(), mid, v)
	}

	return a.send(pendingMessage{typ: message.Response, mid: mid, payload: v})
}

// Close closes the agent, clean inner state and close low-level connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (a *Agent) Close() error {
	if a.GetStatus() == constants.StatusClosed {
		return constants.ErrCloseClosedSession
	}
	a.SetStatus(constants.StatusClosed)

	log.Debugf("Session closed, ID=%d, UID=%d, IP=%s",
		a.Session.ID(), a.Session.UID(), a.Conn.RemoteAddr())

	// prevent closing closed channel
	select {
	case <-a.chDie:
		// expect
	default:
		close(a.chStopWrite)
		close(a.chStopHeartbeat)
		close(a.chStopRead)
		close(a.chDie)
		onSessionClosed(a.Session)
	}

	return a.Conn.Close()
}

// RemoteAddr implementation for session.NetworkEntity interface
// returns the remote network address.
func (a *Agent) RemoteAddr() net.Addr {
	return a.Conn.RemoteAddr()
}

// String, implementation for Stringer interface
func (a *Agent) String() string {
	return fmt.Sprintf("Remote=%s, LastTime=%d", a.Conn.RemoteAddr().String(), a.lastAt)
}

// GetStatus gets the status
func (a *Agent) GetStatus() int32 {
	return atomic.LoadInt32(&a.state)
}

// SetLastAt sets the last at to now
func (a *Agent) SetLastAt() {
	a.lastAt = time.Now().Unix()
}

// SetStatus sets the agent status
func (a *Agent) SetStatus(state int32) {
	atomic.StoreInt32(&a.state, state)
}

func hbdEncode(heartbeatTimeout time.Duration, packetEncoder codec.PacketEncoder) {
	data, err := json.Marshal(map[string]interface{}{
		"code": 200,
		"sys":  map[string]float64{"heartbeat": heartbeatTimeout.Seconds()},
	})
	if err != nil {
		panic(err)
	}

	Hrd, err = packetEncoder.Encode(packet.Handshake, data)
	if err != nil {
		panic(err)
	}

	hbd, err = packetEncoder.Encode(packet.Heartbeat, nil)
	if err != nil {
		panic(err)
	}
}

// Handle handles the messages from and to a client
func (a *Agent) Handle() {
	defer func() {
		a.Close()
		log.Debugf("Session handle goroutine exit, SessionID=%d, UID=%d", a.Session.ID(), a.Session.UID())
	}()
	go a.write()
	go a.read()
	go a.heartbeat()
	select {
	case <-a.chDie: // agent closed signal
		return

	case <-a.appDieChan: // application quit
		return
	}
}

func (a *Agent) heartbeat() {
	ticker := time.NewTicker(a.heartbeatTimeout)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			deadline := time.Now().Add(-2 * a.heartbeatTimeout).Unix()
			if a.lastAt < deadline {
				log.Debugf("Session heartbeat timeout, LastTime=%d, Deadline=%d", a.lastAt, deadline)
				close(a.chDie)
				return
			}
			if _, err := a.Conn.Write(hbd); err != nil {
				close(a.chDie)
				return
			}
		case <-a.chStopHeartbeat:
			return
		}
	}
}

func onSessionClosed(s *session.Session) {
	defer func() {
		if err := recover(); err != nil {
			logger.Log.Errorf("pitaya/onSessionClosed: %v", err)
			logger.Log.Error(util.Stack())
		}
	}()

	if len(s.OnCloseCallbacks) < 1 {
		return
	}

	for _, fn := range s.OnCloseCallbacks {
		fn()
	}
}

func (a *Agent) read() {
	defer func() {
		close(a.chRecv)
	}()
	for {
		select {
		case m := <-a.chRecv:
			a.lastMid = m.LastMid
			util.Pcall(m.Handler, m.Args)
		case <-a.chStopRead:
			return
		}
	}
}

// WriteToChRecv writes to agent recv chan
func (a *Agent) WriteToChRecv(msg *message.UnhandledMessage) {
	a.chRecv <- msg
}

// WriteToChWrite writes to agent write chan
func (a *Agent) WriteToChWrite(data []byte) {
	a.chWrite <- data
}

func (a *Agent) write() {
	// clean func
	defer func() {
		close(a.chSend)
		close(a.chWrite)
	}()

	for {
		select {
		case data := <-a.chWrite:
			// close agent while low-level Conn broken
			if _, err := a.Conn.Write(data); err != nil {
				logger.Log.Error(err.Error())
				return
			}

		case data := <-a.chSend:
			payload, err := util.SerializeOrRaw(a.serializer, data.payload)
			if err != nil {
				logger.Log.Error(err.Error())
				break
			}

			if len(pipeline.Pipeline.Outbound.Handlers) > 0 {
				for _, h := range pipeline.Pipeline.Outbound.Handlers {
					payload, err = h(a.Session, payload)
					if err != nil {
						logger.Log.Debugf("broken pipeline, error: %s", err.Error())
						break
					}
				}
			}

			// construct message and encode
			m := &message.Message{
				Type:  data.typ,
				Data:  payload,
				Route: data.route,
				ID:    data.mid,
			}
			em, err := m.Encode()
			if err != nil {
				logger.Log.Error(err.Error())
				break
			}

			// packet encode
			p, err := a.encoder.Encode(packet.Data, em)
			if err != nil {
				logger.Log.Error(err)
				break
			}
			a.chWrite <- p

		case <-a.chStopWrite:
			return
		}
	}
}
