package component

import (
	"context"
	"github.com/topfreegames/pitaya/v3/pkg/util"
	"math"
	"reflect"
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

func NewMethodProxyContext(origin reflect.Method, args []reflect.Value, fns []MethodProxyFunc) *MethodProxyContext {
	ret := &MethodProxyContext{
		OriginMethod: origin,
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
