package component

import (
	"context"
	"errors"
	"fmt"
	"github.com/topfreegames/pitaya/v3/pkg/constants"
	"github.com/topfreegames/pitaya/v3/pkg/logger"
	"github.com/topfreegames/pitaya/v3/pkg/logger/interfaces"
	"github.com/topfreegames/pitaya/v3/pkg/util"
	"math"
	"reflect"
	"runtime/debug"
	"strconv"
)

type (
	// MethodProxy is a function that will be called when a handler is invoked.
	// origin is the original method of the handler.
	// args are the arguments passed to the method,including:
	// method Receiver at args[0],
	// context.Context at args[1],
	// request struct or raw bytes at args[2] if exists.
	// return proxy func accepts same arguments and returns same values as origin.
	MethodProxy        func(origin reflect.Method, args []reflect.Value) (rets any, err error)
	MethodProxyFunc    func(ctx *MethodProxyContext)
	MethodProxyContext struct {
		index          int8
		handlers       []MethodProxyFunc
		OriginArgs     []reflect.Value
		OriginReceiver reflect.Value
		OriginMethod   reflect.Method
		InCtx          context.Context
		InMsg          any
		OutMsg         any
		OutErr         error
		data           map[string]any
	}
	MethodProxyFuncOption struct {
		Chains map[string][]MethodProxyFunc // method proxy chains, method name -> middleware chain
	}
)

const abortIndex = math.MaxInt8 >> 1

var DefaultHandlerMethodInvoke = util.Pcall

func setUpProxy(proxyFunc []*MethodProxyFuncOption, mn string) (proxy MethodProxy) {
	if len(proxyFunc) == 0 || proxyFunc[0] == nil || len(proxyFunc[0].Chains) == 0 {
		return
	}
	chain := proxyFunc[0].Chains[mn]
	if len(chain) == 0 {
		return
	}
	proxy = func(origin reflect.Method, args []reflect.Value) (rets any, err error) {
		defer func() { // TODO use util.PcallFunc
			if rec := recover(); rec != nil {
				// Try to use logger from context here to help trace error cause
				stackTrace := debug.Stack()
				stackTraceAsRawStringLiteral := strconv.Quote(string(stackTrace))
				log := getLoggerFromArgs(args)
				log.Errorf("panic - pitaya/dispatch/proxy: methodName=%s panicData=%v stackTrace=%s", origin.Name, rec, stackTraceAsRawStringLiteral)

				if s, ok := rec.(string); ok {
					err = errors.New(s)
				} else {
					err = fmt.Errorf("rpc call internal error - %s: %v", origin.Name, rec)
				}
			}
		}()

		ctx := NewMethodProxyContext(origin, args, chain)
		ctx.Next()
		if ctx.IsDone() {
			rets = ctx.OutMsg
			err = ctx.OutErr
			return
		}
		return
	}
	return
}

func getLoggerFromArgs(args []reflect.Value) interfaces.Logger {
	for _, a := range args {
		if !a.IsValid() {
			continue
		}
		if ctx, ok := a.Interface().(context.Context); ok {
			logVal := ctx.Value(constants.LoggerCtxKey)
			if logVal != nil {
				log := logVal.(interfaces.Logger)
				return log
			}
		}
	}
	return logger.Log
}

func NewMethodProxyContext(origin reflect.Method, args []reflect.Value, fns []MethodProxyFunc) *MethodProxyContext {
	ret := &MethodProxyContext{
		OriginMethod: origin,
		OriginArgs:   args,
		data:         make(map[string]any),
		handlers:     fns,
		index:        -1,
	}
	if len(args) > 0 {
		ret.OriginReceiver = args[0]
	}
	if len(args) > 1 {
		ret.InCtx = args[1].Interface().(context.Context)
	}
	if len(args) > 2 {
		ret.InMsg = args[2].Interface()
	}
	return ret
}

func (m *MethodProxyContext) Next() {
	m.index++
	for m.index < int8(len(m.handlers)) {
		m.handlers[m.index](m)
		m.index++
	}
}

func (m *MethodProxyContext) IsDone() bool {
	return m.index >= int8(len(m.handlers)) || m.index >= abortIndex
}

func (m *MethodProxyContext) Abort() {
	m.index = abortIndex
}

func (m *MethodProxyContext) SetData(key string, value any) {
	m.data[key] = value
}

func (m *MethodProxyContext) GetData(key string) any {
	return m.data[key]
}

func (m *MethodProxyContext) GetRequest() any {
	return m.InMsg
}

func (m *MethodProxyContext) GetContext() any {
	return m.InCtx
}

func (m *MethodProxyContext) ResponseWithMsg(msg any) {
	m.OutMsg = msg
	m.Abort()
}

func (m *MethodProxyContext) ResponseWithError(err error) {
	m.OutErr = err
	m.Abort()
}
