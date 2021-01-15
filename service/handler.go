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
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tutumagi/pitaya/acceptor"

	"github.com/google/uuid"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/tutumagi/pitaya/agent"
	"github.com/tutumagi/pitaya/cluster"
	"github.com/tutumagi/pitaya/component"
	"github.com/tutumagi/pitaya/conn/codec"
	"github.com/tutumagi/pitaya/conn/message"
	"github.com/tutumagi/pitaya/conn/packet"
	"github.com/tutumagi/pitaya/constants"
	pcontext "github.com/tutumagi/pitaya/context"
	"github.com/tutumagi/pitaya/docgenerator"
	e "github.com/tutumagi/pitaya/errors"
	"github.com/tutumagi/pitaya/logger"
	"github.com/tutumagi/pitaya/metrics"
	"github.com/tutumagi/pitaya/route"
	"github.com/tutumagi/pitaya/serialize"
	"github.com/tutumagi/pitaya/session"
	"github.com/tutumagi/pitaya/timer"
	"github.com/tutumagi/pitaya/tracing"
)

var (
	handlers    = make(map[string]*component.Handler) // all handler method
	handlerType = "handler"
	// TODO: 后续移到配置文件或采用其他实现方式
	mapFilterRoutes = map[string]bool{
		"baseapp.scene.syncpos": true,
		"baseapp.scene.stopsyncpos": true,
		"baseapp.scene.syncskypos": true,
	}
)

type (
	// HandlerService service
	HandlerService struct {
		appDieChan         chan bool             // die channel app
		chLocalProcess     chan unhandledMessage // channel of messages that will be processed locally
		chRemoteProcess    chan unhandledMessage // channel of messages that will be processed remotely
		MessageChanSize    int
		decoder            codec.PacketDecoder // binary decoder
		encoder            codec.PacketEncoder // binary encoder
		heartbeatTimeout   time.Duration
		messagesBufferSize int
		remoteService      *RemoteService
		serializer         serialize.Serializer          // message serializer
		server             *cluster.Server               // server obj
		services           map[string]*component.Service // all registered service
		messageEncoder     message.Encoder
		metricsReporters   []metrics.Reporter
	}

	unhandledMessage struct {
		ctx   context.Context
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
	messageEncoder message.Encoder,
	metricsReporters []metrics.Reporter,
) *HandlerService {
	h := &HandlerService{
		services:           make(map[string]*component.Service),
		chLocalProcess:     make(chan unhandledMessage, localProcessBufferSize),
		chRemoteProcess:    make(chan unhandledMessage, remoteProcessBufferSize),
		MessageChanSize:    localProcessBufferSize + remoteProcessBufferSize,
		decoder:            packetDecoder,
		encoder:            packetEncoder,
		messagesBufferSize: messagesBufferSize,
		serializer:         serializer,
		heartbeatTimeout:   heartbeatTime,
		appDieChan:         dieChan,
		server:             server,
		remoteService:      remoteService,
		messageEncoder:     messageEncoder,
		metricsReporters:   metricsReporters,
	}

	return h
}

// Dispatch message to corresponding logic handler
func (h *HandlerService) Dispatch(thread int) {
	// TODO: This timer is being stopped multiple times, it probably doesn't need to be stopped here
	defer func() {
		logger.Log.Warnf("Go HandlerService::Dispatch(%d) exit", thread)
		timer.GlobalTicker.Stop()
		if err := recover(); err != nil {
			logger.Log.Warnf("Go HandlerService::Dispatch(%d) exit by err = %v", thread, err)
		}
	}()

	for {
		// Calls to remote servers block calls to local server
		select {
		// 玩家来的消息，在当前app 可以route到，会走到这里
		case lm := <-h.chLocalProcess:
			// logger.Log.Debugf("pitaya.handler Dispatch -> localProcess <0> for SessionID=%d, UID=%s, route=%s", lm.agent.Session.ID(), lm.agent.Session.UID(), lm.msg.Route)
			metrics.ReportMessageProcessDelayFromCtx(lm.ctx, h.metricsReporters, "local")
			h.localProcess(lm.ctx, lm.agent, lm.route, lm.msg)
			// logger.Log.Debugf("pitaya.handler Dispatch -> localProcess <1> for SessionID=%d, UID=%s, route=%s", lm.agent.Session.ID(), lm.agent.Session.UID(), lm.msg.Route)

		// 玩家来的消息，在当前app route不到，会走到这里，去rpc call/post 其他消息
		case rm := <-h.chRemoteProcess:
			// logger.Log.Debugf("pitaya.handler Dispatch -> remoteProcess <0> for SessionID=%d, UID=%s, route=%s", rm.agent.Session.ID(), rm.agent.Session.UID(), rm.msg.Route)
			metrics.ReportMessageProcessDelayFromCtx(rm.ctx, h.metricsReporters, "remote")
			h.remoteService.remoteProcess(rm.ctx, nil, rm.agent, rm.route, rm.msg)
			// logger.Log.Debugf("pitaya.handler Dispatch -> remoteProcess <1> for SessionID=%d, UID=%s, route=%s", rm.agent.Session.ID(), rm.agent.Session.UID(), rm.msg.Route)

		// 收到 rpc call/post 后，处理消息
		case rpcReq := <-h.remoteService.rpcServer.GetUnhandledRequestsChannel():
			// logger.Log.Infof("pitaya.handler Dispatch -> rpc.ProcessSingleMessage <0> for ", zap.Any("rpcReq", rpcReq))
			// logger.Log.Debugf("pitaya.handler Dispatch -> rpc.ProcessSingleMessage <0> for route=%s", rpcReq.Msg.Route)
			h.remoteService.rpcServer.ProcessSingleMessage(rpcReq)
			// logger.Log.Infof("pitaya.handler Dispatch -> rpc.ProcessSingleMessage <1> for ", zap.Any("rpcReq", rpcReq))
			// logger.Log.Debugf("pitaya.handler Dispatch -> rpc.ProcessSingleMessage <1> for route=%s", rpcReq.Msg.Route)

		// timer tick
		case <-timer.GlobalTicker.C: // execute cron task
			timer.Cron()

		// timer create
		case t := <-timer.Manager.ChCreatedTimer: // new Timers
			timer.AddTimer(t)

		// timer close
		case id := <-timer.Manager.ChClosingTimer: // closing Timers
			timer.RemoveTimer(id)
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
func (h *HandlerService) Handle(conn acceptor.PlayerConn) {
	// create a client agent and startup write goroutine
	a := agent.NewAgent(conn, h.decoder, h.encoder, h.serializer, h.heartbeatTimeout, h.messagesBufferSize, h.appDieChan, h.messageEncoder, h.metricsReporters)
	if a.ChRoleMessages == nil {
		a.ChRoleMessages = make(chan agent.UnhandledRoleMessage, h.MessageChanSize)
	}

	// startup agent goroutine
	go a.Handle()
	go h.processGameMessage(a)

	logger.Log.Debugf("New session established: %s", a.String())

	// guarantee agent related resource is destroyed
	defer func() {
		a.Session.Close()
		logger.Log.Debugf("Session read goroutine exit, SessionID=%d, UID=%d", a.Session.ID(), a.Session.UID())
	}()

	for {
		// logger.Log.Debugf("pitaya.handler begin to get nextmessage for SessionID=%d, UID=%s", a.Session.ID(), a.Session.UID())
		msg, err := conn.GetNextMessage()

		if err != nil {
			logger.Log.Errorf("Error reading next available message: %s", err.Error())
			return
		}

		packets, err := h.decoder.Decode(msg)
		if err != nil {
			logger.Log.Errorf("Failed to decode message: %s", err.Error())
			return
		}

		if len(packets) < 1 {
			logger.Log.Warnf("Read no packets, data: %v", msg)
			continue
		}

		// logger.Log.Debugf("pitaya.handler end to decode nextmessage for SessionID=%d, UID=%s", a.Session.ID(), a.Session.UID())

		// process all packet
		for i := range packets {
			if err := h.processPacket(a, packets[i]); err != nil {
				logger.Log.Errorf("Failed to process packet: %s", err.Error())
				return
			}
		}
	}
}

func (h *HandlerService) processPacket(a *agent.Agent, p *packet.Packet) error {
	switch p.Type {
	case packet.Handshake:
		logger.Log.Debug("Received handshake packet")
		// logger.Log.Infof("pitaya.handler end to processPacket :handshake packet for SessionID=%d, UID=%s", a.Session.ID(), a.Session.UID())
		if err := a.SendHandshakeResponse(); err != nil {
			logger.Log.Errorf("Error sending handshake response: %s", err.Error())
			return err
		}
		logger.Log.Debugf("Session handshake Id=%d, Remote=%s", a.Session.ID(), a.RemoteAddr())

		// Parse the json sent with the handshake by the client
		handshakeData := &session.HandshakeData{}
		err := json.Unmarshal(p.Data, handshakeData)
		if err != nil {
			a.SetStatus(constants.StatusClosed)
			return fmt.Errorf("Invalid handshake data. Id=%d", a.Session.ID())
		}

		a.Session.SetHandshakeData(handshakeData)
		a.SetStatus(constants.StatusHandshake)
		err = a.Session.Set(constants.IPVersionKey, a.IPVersion())
		if err != nil {
			logger.Log.Warnf("failed to save ip version on session: %q\n", err)
		}

		logger.Log.Debug("Successfully saved handshake data")

	case packet.HandshakeAck:
		a.SetStatus(constants.StatusWorking)
		logger.Log.Debugf("Receive handshake ACK Id=%d, Remote=%s", a.Session.ID(), a.RemoteAddr())
		// logger.Log.Infof("pitaya.handler end to processPacket :handshake ACK for SessionID=%d, UID=%s", a.Session.ID(), a.Session.UID())

	case packet.Data:
		if a.GetStatus() < constants.StatusWorking {
			return fmt.Errorf("receive data on socket which is not yet ACK, session will be closed immediately, remote=%s",
				a.RemoteAddr().String())
		}

		msg, err := message.Decode(p.Data)
		if err != nil {
			return err
		}

		// logger.Log.Debugf("pitaya.handler begin to processMessage for SessionID=%d, UID=%s, route=%s", a.Session.ID(), a.Session.UID(), msg.Route)
		h.processMessage(a, msg)
		// logger.Log.Debugf("pitaya.handler end to processMessage for SessionID=%d, UID=%s, route=%s", a.Session.ID(), a.Session.UID(), msg.Route)

	case packet.Heartbeat:
		// expected
	}

	a.SetLastAt()
	return nil
}

func (h *HandlerService) processMessage(a *agent.Agent, msg *message.Message) {
	requestID := uuid.New()
	ctx := pcontext.AddToPropagateCtx(context.Background(), constants.StartTimeKey, time.Now().UnixNano())
	ctx = pcontext.AddToPropagateCtx(ctx, constants.RouteKey, msg.Route)
	ctx = pcontext.AddToPropagateCtx(ctx, constants.RequestIDKey, requestID.String())
	tags := opentracing.Tags{
		"local.id":   h.server.ID,
		"span.kind":  "server",
		"msg.type":   strings.ToLower(msg.Type.String()),
		"user.id":    a.Session.UID(),
		"request.id": requestID.String(),
	}
	ctx = tracing.StartSpan(ctx, msg.Route, tags)
	ctx = context.WithValue(ctx, constants.SessionCtxKey, a.Session)

	r, err := route.Decode(msg.Route)
	if err != nil {
		logger.Log.Errorf("Failed to decode route: %s", err.Error())
		a.AnswerWithError(ctx, msg.ID, e.NewError(err, e.ErrBadRequestCode))
		return
	}

	if r.SvType == "" {
		r.SvType = h.server.Type
	}

	// 若走 Dispatch() 协程池， 则打开以下代码 (玩家消息执行顺序无法保证!)
	if _, ok := mapFilterRoutes[msg.Route]; ok {

		//该消息由协程池竞争执行
		message := unhandledMessage{
			ctx:   ctx,
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
				logger.Log.Warnf("request made to another server type but no remoteService running")
			}
		}

	} else {

		//进入用户自己的队列，顺序执行
		message := agent.UnhandledRoleMessage{
			Ctx:   ctx,
			Route: r,
			Msg:   msg,
		}

		a.ChRoleMessages <- message
	}
}

func (h *HandlerService) processGameMessage(a *agent.Agent) {

	uid := a.Session.UID()

	for {
		select {
		case n := <-a.ChRoleMessages:

			if n.Ctx != nil && n.Route != nil && n.Msg != nil {

				m := unhandledMessage{
					ctx:   n.Ctx,
					agent: a,
					route: n.Route,
					msg:   n.Msg,
				}

				uid = a.Session.UID()

				if m.route.SvType == h.server.Type {

					// logger.Log.Debugf("pitaya.handler processGameMessage -> localProcess <0> for SessionID=%d, UID=%s, route=%s", m.agent.Session.ID(), m.agent.Session.UID(), m.msg.Route)
					metrics.ReportMessageProcessDelayFromCtx(m.ctx, h.metricsReporters, "local")
					h.localProcess(m.ctx, m.agent, m.route, m.msg)
					// logger.Log.Debugf("pitaya.handler processGameMessage -> localProcess <1> for SessionID=%d, UID=%s, route=%s", m.agent.Session.ID(), m.agent.Session.UID(), m.msg.Route)

				} else {
					if h.remoteService != nil {

						// logger.Log.Debugf("pitaya.handler processGameMessage -> remoteProcess <0> for SessionID=%d, UID=%s, route=%s", m.agent.Session.ID(), m.agent.Session.UID(), m.msg.Route)
						metrics.ReportMessageProcessDelayFromCtx(m.ctx, h.metricsReporters, "remote")
						h.remoteService.remoteProcess(m.ctx, nil, m.agent, m.route, m.msg)

						// logger.Log.Debugf("pitaya.handler processGameMessage -> remoteProcess <1> for SessionID=%d, UID=%s, route=%s", m.agent.Session.ID(), m.agent.Session.UID(), m.msg.Route)

					} else {
						logger.Log.Warnf("request made to another server type but no remoteService running")
					}
				}
			}
		case <-a.ChAgentDie:
			logger.Log.Warnf("processGameMessage exit. uid = %s", uid)
			return
		}
	}
}

func (h *HandlerService) localProcess(ctx context.Context, a *agent.Agent, route *route.Route, msg *message.Message) {
	var mid uint
	switch msg.Type {
	case message.Request:
		mid = msg.ID
	case message.Notify:
		mid = 0
	}

	ret, err := processHandlerMessage(ctx, route, h.serializer, a.Session, msg.Data, msg.Type, false)
	if msg.Type != message.Notify {
		if err != nil {
			logger.Log.Errorf("Failed to process handler(route:%s) message: %s", route.Short(), err.Error())
			a.AnswerWithError(ctx, mid, err)
		} else {
			err := a.Session.ResponseMID(ctx, mid, ret)
			if err != nil {
				tracing.FinishSpan(ctx, err)
				metrics.ReportTimingFromCtx(ctx, h.metricsReporters, handlerType, err)
			}
		}
	} else {
		metrics.ReportTimingFromCtx(ctx, h.metricsReporters, handlerType, nil)
		tracing.FinishSpan(ctx, err)
	}
}

// // DirectProcessLocalCall 直接 call 当前 server 的方法
// func (h *HandlerService) DirectProcessLocalCall(ctx context.Context, rt *route.Route, reply interface{}, arg interface{}) error {
// 	rsp, err := directRPCLocalCall(ctx, rt, h.serializer, arg)
// 	if err != nil {
// 		return err
// 	}
// 	if reply != nil {
// 		err := h.serializer.Unmarshal(rsp, &reply)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// DumpServices outputs all registered services
func (h *HandlerService) DumpServices() {
	for name, hh := range handlers {
		logger.Log.Infof("registered handler %s, isRawArg: %t, type: %v", name, hh.IsRawArg, hh.MessageType)
	}
}

// Docs returns documentation for handlers
func (h *HandlerService) Docs(getPtrNames bool) (map[string]interface{}, error) {
	if h == nil {
		return map[string]interface{}{}, nil
	}
	return docgenerator.HandlersDocs(h.server.Type, h.services, getPtrNames)
}
