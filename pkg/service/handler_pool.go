package service

import (
	"context"
	"fmt"
	"reflect"

	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/pkg/conn/message"
	"github.com/topfreegames/pitaya/v3/pkg/constants"
	e "github.com/topfreegames/pitaya/v3/pkg/errors"
	"github.com/topfreegames/pitaya/v3/pkg/logger/interfaces"
	"github.com/topfreegames/pitaya/v3/pkg/pipeline"
	"github.com/topfreegames/pitaya/v3/pkg/route"
	"github.com/topfreegames/pitaya/v3/pkg/serialize"
	"github.com/topfreegames/pitaya/v3/pkg/session"
	"github.com/topfreegames/pitaya/v3/pkg/util"
)

// HandlerPool ...
type HandlerPool struct {
	handlers map[string]*component.Handler // all handler method
}

// NewHandlerPool ...
func NewHandlerPool() *HandlerPool {
	return &HandlerPool{
		handlers: make(map[string]*component.Handler),
	}
}

// Register ...
func (h *HandlerPool) Register(serviceName string, name string, handler *component.Handler) {
	h.handlers[fmt.Sprintf("%s.%s", serviceName, name)] = handler
}

// GetHandlers ...
func (h *HandlerPool) GetHandlers() map[string]*component.Handler {
	return h.handlers
}

// ProcessHandlerMessage ...
func (h *HandlerPool) ProcessHandlerMessage(
	ctx context.Context,
	rt *route.Route,
	serializer serialize.Serializer,
	handlerHooks *pipeline.HandlerHooks,
	session session.Session,
	data []byte,
	msgTypeIface interface{},
	remote bool,
) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, constants.SessionCtxKey, session)
	ctx = util.CtxWithDefaultLogger(ctx, rt.String(), session.UID())

	handler, err := h.getHandler(rt)
	if err != nil {
		return nil, e.NewError(err, e.ErrNotFoundCode)
	}

	msgType, err := getMsgType(msgTypeIface)
	if err != nil {
		return nil, e.NewError(err, e.ErrInternalCode)
	}

	logger := ctx.Value(constants.LoggerCtxKey).(interfaces.Logger)
	exit, err := handler.ValidateMessageType(msgType)
	if err != nil && exit {
		return nil, e.NewError(err, e.ErrBadRequestCode)
	} else if err != nil {
		logger.Warnf("invalid message type, error: %s", err.Error())
	}

	// First unmarshal the handler arg that will be passed to
	// both handler and pipeline functions
	arg, err := unmarshalHandlerArg(handler, serializer, data)
	if err != nil {
		return nil, e.NewError(err, e.ErrBadRequestCode)
	}

	ctx, arg, err = handlerHooks.BeforeHandler.ExecuteBeforePipeline(ctx, arg)
	if err != nil {
		return nil, err
	}

	logger.Debugf("SID=%d, Data=%s", session.ID(), data)
	args := []reflect.Value{handler.Receiver, reflect.ValueOf(ctx)}
	if arg != nil {
		args = append(args, reflect.ValueOf(arg))
	}

	resp, err := util.Pcall(handler.Method, args)
	if remote && msgType == message.Notify {
		// This is a special case and should only happen with nats rpc client
		// because we used nats request we have to answer to it or else a timeout
		// will happen in the caller server and will be returned to the client
		// the reason why we don't just Publish is to keep track of failed rpc requests
		// with timeouts, maybe we can improve this flow
		resp = []byte("ack")
	}

	resp, err = handlerHooks.AfterHandler.ExecuteAfterPipeline(ctx, resp, err)
	if err != nil {
		return nil, err
	}

	ret, err := serializeReturn(serializer, resp)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (h *HandlerPool) getHandler(rt *route.Route) (*component.Handler, error) {
	handler, ok := h.handlers[rt.Short()]
	if !ok {
		e := fmt.Errorf("pitaya/handler: %s not found", rt.String())
		return nil, e
	}
	return handler, nil

}
