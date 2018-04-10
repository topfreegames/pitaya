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
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/component"
)

type MyComp struct {
	component.Base
	running bool
}

func (m *MyComp) Init() {
	m.running = true
}

func (m *MyComp) Shutdown() {
	m.running = false
}

func resetComps() {
	handlerComp = make([]regComp, 0)
	remoteComp = make([]regComp, 0)
}

func TestRegister(t *testing.T) {
	resetComps()
	b := &component.Base{}
	Register(b)
	assert.Equal(t, 1, len(handlerComp))
	assert.Equal(t, regComp{b, nil}, handlerComp[0])
}

func TestRegisterRemote(t *testing.T) {
	resetComps()
	b := &component.Base{}
	RegisterRemote(b)
	assert.Equal(t, 1, len(remoteComp))
	assert.Equal(t, regComp{b, nil}, remoteComp[0])
}

func TestStartupComponents(t *testing.T) {
	initApp()
	resetComps()
	Configure(true, "testtype", Standalone, map[string]string{}, viper.New())

	Register(&MyComp{})
	RegisterRemote(&MyComp{})
	startupComponents()
	assert.Equal(t, true, handlerComp[0].comp.(*MyComp).running)
}

func TestShutdownComponents(t *testing.T) {
	resetComps()
	initApp()
	Configure(true, "testtype", Standalone, map[string]string{}, viper.New())

	Register(&MyComp{})
	RegisterRemote(&MyComp{})
	startupComponents()

	shutdownComponents()
	assert.Equal(t, false, handlerComp[0].comp.(*MyComp).running)
}
