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
	"fmt"
	"net"
	"time"

	"github.com/topfreegames/pitaya/agent"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/internal/packet"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/serialize"
	"github.com/topfreegames/pitaya/timer"
)

var (
	handlers = make(map[string]*component.Handler) // all handler method
	log      = logger.Log
)

type (
	// HandlerService service
	HandlerService struct {
		appDieChan         chan bool             // die channel app
		chLocalProcess     chan unhandledMessage // channel of messages that will be processed locally
		chRemoteProcess    chan unhandledMessage // channel of messages that will be processed remotely
		decoder            codec.PacketDecoder   // binary decoder
		encoder            codec.PacketEncoder   // binary encoder
		heartbeatTimeout   time.Duration
		messagesBufferSize int
		remoteService      *RemoteService
		serializer         serialize.Serializer          // message serializer
		server             *cluster.Server               // server obj
		services           map[string]*component.Service // all registered service
	}

	unhandledMessage struct {
		agent *agent.Agent
		route *route.Route
		msg   *message.Message
	}
)

// NewHandlerService creates and returns a new handler service
func NewHandlerService(
	dieChan chan bool,
	packetDecoder codec.PacketDecoder,
	packetEncoder codec.PacketEncoder,
	serializer serialize.Serializer,
	heartbeatTime time.Duration,
	messagesBufferSize,
	localProcessBufferSize,
	remoteProcessBufferSize int,
	server *cluster.Server,
	remoteService *RemoteService,
) *HandlerService {
	h := &HandlerService{
		services:           make(map[string]*component.Service),
		chLocalProcess:     make(chan unhandledMessage, localProcessBufferSize),
		chRemoteProcess:    make(chan unhandledMessage, remoteProcessBufferSize),
		decoder:            packetDecoder,
		encoder:            packetEncoder,
		messagesBufferSize: messagesBufferSize,
		serializer:         serializer,
		heartbeatTimeout:   heartbeatTime,
		appDieChan:         dieChan,
		server:             server,
		remoteService:      remoteService,
	}

	return h
}

// Dispatch message to corresponding logic handler
func (h *HandlerService) Dispatch(thread int) {
	// close chLocalProcess & chCloseSession when application quit
	defer timer.GlobalTicker.Stop()

	// handle packet that sent to chLocalProcess
	for {
		select {
		case lm := <-h.chLocalProcess:
			h.localProcess(lm.agent, lm.route, lm.msg)
		case rm := <-h.chRemoteProcess:
			h.remoteService.remoteProcess(nil, rm.agent, rm.route, rm.msg)

		case <-timer.GlobalTicker.C: // execute cron task
			timer.Cron()

		case t := <-timer.Manager.ChCreatedTimer: // new Timers
			timer.AddTimer(t)

		case id := <-timer.Manager.ChClosingTimer: // closing Timers
			timer.RemoveTimer(id)

		case <-h.appDieChan: // application quit signal
			return
		}
	}
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
	a := agent.NewAgent(conn, h.decoder, h.encoder, h.serializer, h.heartbeatTimeout, h.messagesBufferSize, h.appDieChan)

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

		// (warning): decoder use slice for performance, packet data should be copy before next Decode
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
		if err := a.SendHandshakeResponse(); err != nil {
			return err
		}
		a.SetStatus(constants.StatusHandshake)
		log.Debugf("Session handshake Id=%d, Remote=%s", a.Session.ID(), a.RemoteAddr())

	case packet.HandshakeAck:
		a.SetStatus(constants.StatusWorking)
		log.Debugf("Receive handshake ACK Id=%d, Remote=%s", a.Session.ID(), a.RemoteAddr())

	case packet.Data:
		if a.GetStatus() < constants.StatusWorking {
			return fmt.Errorf("receive data on socket which not yet ACK, session will be closed immediately, remote=%s",
				a.RemoteAddr().String())
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

	message := unhandledMessage{
		agent: a,
		route: r,
		msg:   msg,
	}
	if r.SvType == h.server.Type {
		h.chLocalProcess <- message
	} else {
		if h.remoteService != nil {
			h.chRemoteProcess <- message
		} else {
			log.Warnf("request made to another server type but no remoteService running")
		}
	}
}

func (h *HandlerService) localProcess(a *agent.Agent, route *route.Route, msg *message.Message) {
	var mid uint
	switch msg.Type {
	case message.Request:
		mid = msg.ID
	case message.Notify:
		mid = 0
	}

	ret, err := processHandlerMessage(route, h.serializer, a.Srv, a.Session, msg.Data, msg.Type, false)
	if err != nil {
		log.Error(err)
		a.AnswerWithError(mid, err)
	} else {
		a.Session.ResponseMID(mid, ret)
	}
}

// DumpServices outputs all registered services
func (h *HandlerService) DumpServices() {
	for name := range handlers {
		log.Infof("registered handler %s, isRawArg: %s", name, handlers[name].IsRawArg)
	}
}
