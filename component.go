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
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/logger"
)

var (
	handlerComp = make([]regComp, 0)
	remoteComp  = make([]regComp, 0)
)

type regComp struct {
	comp component.Component
	opts []component.Option
}

// Register register a component with options
func Register(c component.Component, options ...component.Option) {
	handlerComp = append(handlerComp, regComp{c, options})
}

// RegisterRemote register a remote component with options
func RegisterRemote(c component.Component, options ...component.Option) {
	remoteComp = append(remoteComp, regComp{c, options})
}

func startupComponents() {
	// component initialize hooks
	for _, c := range handlerComp {
		c.comp.Init()
	}

	// component after initialize hooks
	for _, c := range handlerComp {
		c.comp.AfterInit()
	}

	// register all components
	for _, c := range handlerComp {
		if err := handlerService.Register(c.comp, c.opts); err != nil {
			logger.Log.Errorf("Failed to register handler: %s", err.Error())
		}
	}

	// register all remote components
	for _, c := range remoteComp {
		if remoteService == nil {
			logger.Log.Warn("registered a remote component but remoteService is not running! skipping...")
		} else {
			if err := remoteService.Register(c.comp, c.opts); err != nil {
				logger.Log.Errorf("Failed to register remote: %s", err.Error())
			}
		}
	}

	handlerService.DumpServices()
	if remoteService != nil {
		remoteService.DumpServices()
	}
}

func shutdownComponents() {
	// reverse call `BeforeShutdown` hooks
	length := len(handlerComp)
	for i := length - 1; i >= 0; i-- {
		handlerComp[i].comp.BeforeShutdown()
	}

	// reverse call `Shutdown` hooks
	for i := length - 1; i >= 0; i-- {
		handlerComp[i].comp.Shutdown()
	}

	length = len(remoteComp)
	for i := length - 1; i >= 0; i-- {
		remoteComp[i].comp.BeforeShutdown()
	}

	// reverse call `Shutdown` hooks
	for i := length - 1; i >= 0; i-- {
		remoteComp[i].comp.Shutdown()
	}
}
