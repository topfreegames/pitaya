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
	"github.com/topfreegames/pitaya/v2/component"
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

func TestRegister(t *testing.T) {
	config := viper.New()
	app := NewDefaultApp(true, "testtype", Cluster, map[string]string{}, config).(*App)
	b := &component.Base{}
	app.Register(b)
	assert.Equal(t, 1, len(app.handlerComp))
	assert.Equal(t, regComp{b, nil}, app.handlerComp[0])
}

func TestRegisterRemote(t *testing.T) {
	config := viper.New()
	app := NewDefaultApp(true, "testtype", Cluster, map[string]string{}, config).(*App)
	before := app.remoteComp
	b := &component.Base{}
	app.RegisterRemote(b)
	assert.Equal(t, len(before)+1, len(app.remoteComp))
	assert.Equal(t, regComp{b, nil}, app.remoteComp[len(before)])
}

func TestStartupComponents(t *testing.T) {
	app := NewDefaultApp(true, "testtype", Standalone, map[string]string{}, viper.New()).(*App)

	app.Register(&MyComp{})
	app.RegisterRemote(&MyComp{})
	app.startupComponents()
	assert.Equal(t, true, app.handlerComp[0].comp.(*MyComp).running)
}

func TestShutdownComponents(t *testing.T) {
	app := NewDefaultApp(true, "testtype", Standalone, map[string]string{}, viper.New()).(*App)

	app.Register(&MyComp{})
	app.RegisterRemote(&MyComp{})
	app.startupComponents()

	app.shutdownComponents()
	assert.Equal(t, false, app.handlerComp[0].comp.(*MyComp).running)
}
