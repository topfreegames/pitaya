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
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/internal/packet"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/session"
)

// Unhandled message buffer size
const packetBacklog = 1024
const funcBacklog = 1 << 8

var (
	// handler service singleton
	handler = newHandlerService()

	// serialized data
	hrd []byte // handshake response data
	hbd []byte // heartbeat packet data
)

func hbdEncode() {
	data, err := json.Marshal(map[string]interface{}{
		"code": 200,
		"sys":  map[string]float64{"heartbeat": app.heartbeat.Seconds()},
	})
	if err != nil {
		panic(err)
	}

	hrd, err = app.packetEncoder.Encode(packet.Handshake, data)
	if err != nil {
		panic(err)
	}

	hbd, err = app.packetEncoder.Encode(packet.Heartbeat, nil)
	if err != nil {
		panic(err)
	}
}

type (
	handlerService struct {
		services   map[string]*component.Service // all registered service
		handlers   map[string]*component.Handler // all handler method
		chFunction chan func()                   // function that called in logic gorontine
	}

	unhandledMessage struct {
		lastMid uint
		handler reflect.Method
		args    []reflect.Value
	}
)

func newHandlerService() *handlerService {
	h := &handlerService{
		services:   make(map[string]*component.Service),
		handlers:   make(map[string]*component.Handler),
		chFunction: make(chan func(), funcBacklog),
	}

	return h
}

// call handler with protected
func pcall(method reflect.Method, args []reflect.Value) {
	defer func() {
		if err := recover(); err != nil {
			logger.Log.Errorf("pitaya/dispatch: %v", err)
			logger.Log.Error(stack())
		}
	}()

	if r := method.Func.Call(args); len(r) > 0 {
		if err := r[0].Interface(); err != nil {
			logger.Log.Error(err.(error).Error())
		}
	}
}

// call handler with protected
func pinvoke(fn func()) {
	defer func() {
		if err := recover(); err != nil {
			logger.Log.Errorf("pitaya/invoke: %v", err)
			logger.Log.Error(stack())
		}
	}()

	fn()
}

func onSessionClosed(s *session.Session) {
	defer func() {
		if err := recover(); err != nil {
			logger.Log.Errorf("pitaya/onSessionClosed: %v", err)
			logger.Log.Error(stack())
		}
	}()

	if len(s.OnCloseCallbacks) < 1 {
		return
	}

	for _, fn := range s.OnCloseCallbacks {
		fn()
	}
}

// dispatch message to corresponding logic handler
func (h *handlerService) dispatch() {
	// close chLocalProcess & chCloseSession when application quit
	defer func() {
		globalTicker.Stop()
	}()

	// handle packet that sent to chLocalProcess
	for {
		select {
		case fn := <-h.chFunction:
			pinvoke(fn)

		case <-globalTicker.C: // execute cron task
			cron()

		case t := <-timerManager.chCreatedTimer: // new timers
			timerManager.timers[t.id] = t

		case id := <-timerManager.chClosingTimer: // closing timers
			delete(timerManager.timers, id)

		case <-app.dieChan: // application quit signal
			return
		}
	}
}

func (h *handlerService) register(comp component.Component, opts []component.Option) error {
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
		h.handlers[fmt.Sprintf("%s.%s", s.Name, name)] = handler
	}
	return nil
}

func (h *handlerService) handle(conn net.Conn) {
	// create a client agent and startup write gorontine
	agent := newAgent(conn)

	// startup agent goroutine
	go agent.handle()

	logger.Log.Debugf("New session established: %s", agent.String())

	// guarantee agent related resource be destroyed
	defer func() {
		agent.session.Close()
		logger.Log.Debugf("Session read goroutine exit, SessionID=%d, UID=%d", agent.session.ID(), agent.session.UID())
	}()

	// read loop
	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			logger.Log.Debugf("Read message error: %s, session will be closed immediately", err.Error())
			return
		}

		// TODO(warning): decoder use slice for performance, packet data should be copy before next Decode
		packets, err := agent.decoder.Decode(buf[:n])
		if err != nil {
			logger.Log.Error(err.Error())
			return
		}

		if len(packets) < 1 {
			continue
		}

		// process all packet
		for i := range packets {
			if err := h.processPacket(agent, packets[i]); err != nil {
				logger.Log.Error(err.Error())
				return
			}
		}
	}
}

func (h *handlerService) processPacket(agent *agent, p *packet.Packet) error {
	switch p.Type {
	case packet.Handshake:
		if _, err := agent.conn.Write(hrd); err != nil {
			return err
		}

		agent.setStatus(statusHandshake)
		logger.Log.Debugf("Session handshake Id=%d, Remote=%s", agent.session.ID(), agent.conn.RemoteAddr())

	case packet.HandshakeAck:
		agent.setStatus(statusWorking)
		logger.Log.Debugf("Receive handshake ACK Id=%d, Remote=%s", agent.session.ID(), agent.conn.RemoteAddr())

	case packet.Data:
		if agent.status() < statusWorking {
			return fmt.Errorf("receive data on socket which not yet ACK, session will be closed immediately, remote=%s",
				agent.conn.RemoteAddr().String())
		}

		msg, err := message.Decode(p.Data)
		if err != nil {
			return err
		}
		h.processMessage(agent, msg)

	case packet.Heartbeat:
		// expected
	}

	agent.lastAt = time.Now().Unix()
	return nil
}

func (h *handlerService) processMessage(agent *agent, msg *message.Message) {
	r, err := route.Decode(msg.Route)

	if err != nil {
		log.Error(err.Error())
		return
	}

	if r.SvType == "" {
		r.SvType = app.server.Type
	}

	if r.SvType == app.server.Type {
		h.localProcess(agent, r, msg)
	} else {
		h.remoteProcess(agent, r, msg)
	}
}

func (h *handlerService) remoteProcess(agent *agent, route *route.Route, msg *message.Message) {
	var res []byte
	var err error
	if res, err = remoteCall(protos.RPCType_Sys, route, agent.session, msg); err != nil {
		// TODO: we should probably return the error to the client
		log.Errorf(err.Error())
		return
	}
	// TODO CAMILA return to the client here (send to a.chWrite)
	// TODO remove
	fmt.Printf("cool, got %s\n", res)
	agent.chWrite <- res
}

func (h *handlerService) localProcess(agent *agent, route *route.Route, msg *message.Message) {
	var lastMid uint
	switch msg.Type {
	case message.Request:
		lastMid = msg.ID
	case message.Notify:
		lastMid = 0
	}

	handler, ok := h.handlers[fmt.Sprintf("%s.%s", route.Service, route.Method)]
	if !ok {
		logger.Log.Warnf("pitaya/handler: %s not found(forgot registered?)", msg.Route)
		return
	}

	var payload = msg.Data
	var err error
	if len(Pipeline.Inbound.handlers) > 0 {
		for _, h := range Pipeline.Inbound.handlers {
			payload, err = h(agent.session, payload)
			if err != nil {
				logger.Log.Errorf("pitaya/handler: broken pipeline: %s", err.Error())
				return
			}
		}
	}

	var data interface{}
	if handler.IsRawArg {
		data = payload
	} else {
		data = reflect.New(handler.Type.Elem()).Interface()
		err := app.serializer.Unmarshal(payload, data)
		if err != nil {
			logger.Log.Error("deserialize error", err.Error())
			return
		}
	}

	log.Debugf("UID=%d, Message={%s}, Data=%+v", agent.session.UID(), msg.String(), data)

	args := []reflect.Value{handler.Receiver, agent.srv, reflect.ValueOf(data)}
	agent.chRecv <- unhandledMessage{lastMid, handler.Method, args}

}

// DumpServices outputs all registered services
func (h *handlerService) DumpServices() {
	for name := range h.handlers {
		logger.Log.Infof("registered service %s", name)
	}
}
