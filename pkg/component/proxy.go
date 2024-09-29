package component

import (
	"github.com/topfreegames/pitaya/v3/pkg/util"
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
	MethodProxy        = func(origin reflect.Method, args []reflect.Value) (rets any, err error)
	MethodProxyFunc    = func(ctx *MethodProxyContext) MethodProxy
	MethodProxyContext struct {
		State bool
	}
	MethodProxyFuncOption struct {
		Chains map[string][]MethodProxyFunc // method proxy chains, method name -> middleware chain
	}
)

var DefaultHandlerMethodInvoke = util.Pcall

func (m *MethodProxyContext) Next() {
	m.State = true
}

func (m *MethodProxyContext) IsDone() bool {
	return m.State
}

func (m *MethodProxyContext) Abort() {
	m.State = false
}
