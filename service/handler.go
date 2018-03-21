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

package service

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/topfreegames/pitaya/agent"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/internal/packet"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/pipeline"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/serialize"
	"github.com/topfreegames/pitaya/timer"
	"github.com/topfreegames/pitaya/util"
)

// Unhandled message buffer size
const packetBacklog = 1024
const funcBacklog = 1 << 8

var log = logger.Log

var (
	handlers = make(map[string]*component.Handler) // all handler method
)

type (
	// HandlerService service
	HandlerService struct {
		services         map[string]*component.Service // all registered service
		chFunction       chan func()                   // function that called in logic gorontine
		appDieChan       chan bool                     // die channel app
		decoder          codec.PacketDecoder           // binary decoder
		encoder          codec.PacketEncoder           // binary encoder
		serializer       serialize.Serializer          // message serializer
		heartbeatTimeout time.Duration
		server           *cluster.Server // server obj
		remoteService    *RemoteService
	}
)

// NewHandlerService creates and returns a new handler service
func NewHandlerService(
	dieChan chan bool,
	packetDecoder codec.PacketDecoder,
	packetEncoder codec.PacketEncoder,
	serializer serialize.Serializer,
	heartbeatTime time.Duration,
	server *cluster.Server,
	remoteService *RemoteService,
) *HandlerService {
	h := &HandlerService{
		services:         make(map[string]*component.Service),
		chFunction:       make(chan func(), funcBacklog),
		decoder:          packetDecoder,
		encoder:          packetEncoder,
		serializer:       serializer,
		heartbeatTimeout: heartbeatTime,
		appDieChan:       dieChan,
		server:           server,
		remoteService:    remoteService,
	}

	return h
}

// Dispatch message to corresponding logic handler
func (h *HandlerService) Dispatch() {
	// close chLocalProcess & chCloseSession when application quit
	defer timer.GlobalTicker.Stop()

	// handle packet that sent to chLocalProcess
	for {
		select {
		case fn := <-h.chFunction:
			util.Pinvoke(fn)

		case <-timer.GlobalTicker.C: // execute cron task
			timer.Cron()

		case t := <-timer.Manager.ChCreatedTimer: // new Timers
			timer.Manager.Timers[t.ID] = t

		case id := <-timer.Manager.ChClosingTimer: // closing Timers
			delete(timer.Manager.Timers, id)

		case <-h.appDieChan: // application quit signal
			return
		}
	}
}

// Invoke invokes function in main logic goroutine
func (h *HandlerService) Invoke(fn func()) {
	h.chFunction <- fn
}

// Register registers components
func (h *HandlerService) Register(comp component.Component, opts []component.Option) error {
	s := component.NewService(comp, opts)

	if _, ok := h.services[s.Name]; ok {
		return fmt.Errorf("handler: service already defined: %s", s.Name)
	}

	if err := s.ExtractHandler(); err != nil {
		return err
	}

	// register all handlers
	h.services[s.Name] = s
	for name, handler := range s.Handlers {
		handlers[fmt.Sprintf("%s.%s", s.Name, name)] = handler
	}
	return nil
}

// Handle handles messages from a conn
func (h *HandlerService) Handle(conn net.Conn) {
	// create a client agent and startup write gorontine
	a := agent.NewAgent(conn, h.decoder, h.encoder, h.serializer, h.heartbeatTimeout, h.appDieChan)

	// startup agent goroutine
	go a.Handle()

	log.Debugf("New session established: %s", a.String())

	// guarantee agent related resource be destroyed
	defer func() {
		a.Session.Close()
		log.Debugf("Session read goroutine exit, SessionID=%d, UID=%d", a.Session.ID(), a.Session.UID())
	}()

	// read loop
	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Debugf("Read message error: %s, session will be closed immediately", err.Error())
			return
		}

		// TODO(warning): decoder use slice for performance, packet data should be copy before next Decode
		packets, err := h.decoder.Decode(buf[:n])
		if err != nil {
			log.Error(err.Error())
			return
		}

		if len(packets) < 1 {
			continue
		}

		// process all packet
		for i := range packets {
			if err := h.processPacket(a, packets[i]); err != nil {
				log.Error(err.Error())
				return
			}
		}
	}
}

func (h *HandlerService) processPacket(a *agent.Agent, p *packet.Packet) error {
	switch p.Type {
	case packet.Handshake:
		if _, err := a.Conn.Write(agent.Hrd); err != nil {
			return err
		}
		a.SetStatus(constants.StatusHandshake)
		log.Debugf("Session handshake Id=%d, Remote=%s", a.Session.ID(), a.Conn.RemoteAddr())

	case packet.HandshakeAck:
		a.SetStatus(constants.StatusWorking)
		log.Debugf("Receive handshake ACK Id=%d, Remote=%s", a.Session.ID(), a.Conn.RemoteAddr())

	case packet.Data:
		if a.GetStatus() < constants.StatusWorking {
			return fmt.Errorf("receive data on socket which not yet ACK, session will be closed immediately, remote=%s",
				a.Conn.RemoteAddr().String())
		}

		msg, err := message.Decode(p.Data)
		if err != nil {
			return err
		}
		h.processMessage(a, msg)

	case packet.Heartbeat:
		// expected
	}

	a.SetLastAt()
	return nil
}

func (h *HandlerService) processMessage(a *agent.Agent, msg *message.Message) {
	r, err := route.Decode(msg.Route)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if r.SvType == "" {
		r.SvType = h.server.Type
	}

	if r.SvType == h.server.Type {
		h.localProcess(a, r, msg)
	} else {
		if h.remoteService != nil {
			h.remoteService.remoteProcess(a, r, msg)
		} else {
			log.Warnf("request made to another server type but no remoteService running")
		}
	}
}

func (h *HandlerService) localProcess(a *agent.Agent, route *route.Route, msg *message.Message) {
	var lastMid uint
	switch msg.Type {
	case message.Request:
		lastMid = msg.ID
	case message.Notify:
		lastMid = 0
	}

	handler, ok := handlers[fmt.Sprintf("%s.%s", route.Service, route.Method)]
	if !ok {
		e := fmt.Sprintf("pitaya/handler: %s not found", msg.Route)
		log.Warn(e)
		agent.AnswerWithError(a, msg.ID, errors.New(e))
		return
	}

	var payload = msg.Data
	var err error
	if len(pipeline.BeforeHandler.Handlers) > 0 {
		for _, h := range pipeline.BeforeHandler.Handlers {
			payload, err = h(a.Session, payload)
			if err != nil {
				log.Errorf("pitaya/handler: broken pipeline: %s", err.Error())
				agent.AnswerWithError(a, msg.ID, err)
				return
			}
		}
	}

	var data interface{}
	if handler.IsRawArg {
		data = payload
	} else {
		data = reflect.New(handler.Type.Elem()).Interface()
		err := h.serializer.Unmarshal(payload, data)
		if err != nil {
			e := fmt.Errorf("deserialize error: %s", err.Error())
			log.Warn(e)
			agent.AnswerWithError(a, msg.ID, e)
			return
		}
	}

	log.Debugf("UID=%d, Message={%s}, Data=%+v", a.Session.UID(), msg.String(), data)

	args := []reflect.Value{handler.Receiver, a.Srv, reflect.ValueOf(data)}
	a.WriteToChRecv(&message.UnhandledMessage{
		LastMid: lastMid,
		Handler: handler.Method,
		Args:    args,
	})
}

// DumpServices outputs all registered services
func (h *HandlerService) DumpServices() {
	for name := range handlers {
		log.Infof("registered handler %s", name)
	}
}
