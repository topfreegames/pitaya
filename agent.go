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

package pitaya

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/internal/packet"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/session"
)

const (
	agentWriteBacklog = 16
)

var (
	// ErrBrokenPipe represents the low-level connection has broken.
	ErrBrokenPipe = errors.New("broken low-level pipe")
	// ErrBufferExceed indicates that the current session buffer is full and
	// can not receive more data.
	ErrBufferExceed = errors.New("session send buffer exceed")
)

type (
	// Agent corresponding a user, used for store raw conn information
	agent struct {
		// regular agent member
		session         *session.Session      // session
		conn            net.Conn              // low-level conn fd
		lastMid         uint                  // last message id
		state           int32                 // current agent state
		chDie           chan struct{}         // wait for close
		chSend          chan pendingMessage   // push message queue
		chRecv          chan unhandledMessage // unhandledMessages
		chWrite         chan []byte           // write message to the clients
		chStopWrite     chan struct{}         // stop writing messages
		chStopHeartbeat chan struct{}         // stop heartbeats
		chStopRead      chan struct{}         //stop reading
		lastAt          int64                 // last heartbeat unix time stamp
		decoder         codec.PacketDecoder   // binary decoder

		srv reflect.Value // cached session reflect.Value
	}

	pendingMessage struct {
		typ     message.Type // message type
		route   string       // message route(push)
		mid     uint         // response message id(response)
		payload interface{}  // payload
	}
)

// Create new agent instance
func newAgent(conn net.Conn) *agent {
	a := &agent{
		conn:            conn,
		state:           statusStart,
		chDie:           make(chan struct{}),
		chStopWrite:     make(chan struct{}),
		chStopHeartbeat: make(chan struct{}),
		chStopRead:      make(chan struct{}),
		chWrite:         make(chan []byte, agentWriteBacklog),
		lastAt:          time.Now().Unix(),
		chSend:          make(chan pendingMessage, agentWriteBacklog),
		chRecv:          make(chan unhandledMessage),
		decoder:         app.packetDecoder,
	}

	// binding session
	s := session.New(a)
	a.session = s
	a.srv = reflect.ValueOf(s)

	return a
}

func (a *agent) send(m pendingMessage) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = ErrBrokenPipe
		}
	}()
	a.chSend <- m
	return
}

func (a *agent) MID() uint {
	return a.lastMid
}

// Push, implementation for session.NetworkEntity interface
func (a *agent) Push(route string, v interface{}) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}

	if len(a.chSend) >= agentWriteBacklog {
		return ErrBufferExceed
	}

	switch d := v.(type) {
	case []byte:
		logger.Log.Debugf("Type=Push, ID=%d, UID=%d, Route=%s, Data=%dbytes",
			a.session.ID(), a.session.UID(), route, len(d))
	default:
		logger.Log.Debugf("Type=Push, ID=%d, UID=%d, Route=%s, Data=%+v",
			a.session.ID(), a.session.UID(), route, v)
	}

	return a.send(pendingMessage{typ: message.Push, route: route, payload: v})
}

// Response, implementation for session.NetworkEntity interface
// Response message to session
func (a *agent) Response(v interface{}) error {
	return a.ResponseMID(a.lastMid, v)
}

// Response, implementation for session.NetworkEntity interface
// Response message to session
func (a *agent) ResponseMID(mid uint, v interface{}) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}

	if mid <= 0 {
		return ErrSessionOnNotify
	}

	if len(a.chSend) >= agentWriteBacklog {
		return ErrBufferExceed
	}

	switch d := v.(type) {
	case []byte:
		logger.Log.Debugf("Type=Response, ID=%d, UID=%d, MID=%d, Data=%dbytes",
			a.session.ID(), a.session.UID(), mid, len(d))
	default:
		logger.Log.Infof("Type=Response, ID=%d, UID=%d, MID=%d, Data=%+v",
			a.session.ID(), a.session.UID(), mid, v)
	}

	return a.send(pendingMessage{typ: message.Response, mid: mid, payload: v})
}

// Close, implementation for session.NetworkEntity interface
// Close closes the agent, clean inner state and close low-level connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (a *agent) Close() error {
	if a.status() == statusClosed {
		return ErrCloseClosedSession
	}
	a.setStatus(statusClosed)

	logger.Log.Debugf("Session closed, ID=%d, UID=%d, IP=%s",
		a.session.ID(), a.session.UID(), a.conn.RemoteAddr())

	// prevent closing closed channel
	select {
	case <-a.chDie:
		// expect
	default:
		close(a.chStopWrite)
		close(a.chStopHeartbeat)
		close(a.chStopRead)
		close(a.chDie)
		onSessionClosed(a.session)
	}

	return a.conn.Close()
}

// RemoteAddr, implementation for session.NetworkEntity interface
// returns the remote network address.
func (a *agent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

// String, implementation for Stringer interface
func (a *agent) String() string {
	return fmt.Sprintf("Remote=%s, LastTime=%d", a.conn.RemoteAddr().String(), a.lastAt)
}

func (a *agent) status() int32 {
	return atomic.LoadInt32(&a.state)
}

func (a *agent) setStatus(state int32) {
	atomic.StoreInt32(&a.state, state)
}

func (a *agent) handle() {
	defer func() {
		a.Close()
		log.Debugf("Session handle goroutine exit, SessionID=%d, UID=%d", a.session.ID(), a.session.UID())
	}()
	go a.write()
	go a.read()
	go a.heartbeat()
	select {
	case <-a.chDie: // agent closed signal
		return

	case <-app.dieChan: // application quit
		return
	}
}

func (a *agent) heartbeat() {
	ticker := time.NewTicker(app.heartbeat)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			deadline := time.Now().Add(-2 * app.heartbeat).Unix()
			if a.lastAt < deadline {
				log.Debugf("Session heartbeat timeout, LastTime=%d, Deadline=%d", a.lastAt, deadline)
				close(a.chDie)
				return
			}
			if _, err := a.conn.Write(hbd); err != nil {
				close(a.chDie)
				return
			}
		case <-a.chStopHeartbeat:
			return
		}
	}
}

func (a *agent) read() {
	defer func() {
		close(a.chRecv)
	}()
	for {
		select {
		case m := <-a.chRecv:
			a.lastMid = m.lastMid
			pcall(m.handler, m.args)
		case <-a.chStopRead:
			return
		}
	}
}

func (a *agent) write() {
	// clean func
	defer func() {
		close(a.chSend)
		close(a.chWrite)
	}()

	for {
		select {
		case data := <-a.chWrite:
			// close agent while low-level conn broken
			if _, err := a.conn.Write(data); err != nil {
				logger.Log.Error(err.Error())
				return
			}

		case data := <-a.chSend:
			payload, err := serializeOrRaw(data.payload)
			if err != nil {
				logger.Log.Error(err.Error())
				break
			}

			if len(Pipeline.Outbound.handlers) > 0 {
				for _, h := range Pipeline.Outbound.handlers {
					payload, err = h(a.session, payload)
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
			p, err := app.packetEncoder.Encode(packet.Data, em)
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
