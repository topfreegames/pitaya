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
	"github.com/topfreegames/pitaya/interfaces"
)

type MyMod struct {
	component.Base
	running bool
}

func (m *MyMod) Init() error {
	m.running = true
	return nil
}

func (m *MyMod) Shutdown() error {
	m.running = false
	return nil
}

func resetModules() {
	modules = make(map[string]interfaces.Module)
}

func TestRegisterModule(t *testing.T) {
	resetModules()
	b := &MyMod{}
	err := RegisterModule(b, "mod")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(modules))
	assert.Equal(t, b, modules["mod"])
	err = RegisterModule(b, "mod")
	assert.Error(t, err)
}

func TestGetModule(t *testing.T) {
	resetModules()
	b := &MyMod{}
	RegisterModule(b, "mod")
	m, err := GetModule("mod")
	assert.NoError(t, err)
	assert.Equal(t, b, m)

	m, err = GetModule("mmm")
	assert.Error(t, err)
}

func TestStartupModules(t *testing.T) {
	initApp()
	resetModules()
	Configure(true, "testtype", Standalone, map[string]string{}, viper.New())

	RegisterModule(&MyMod{}, "bla")
	startModules()
	assert.Equal(t, true, modules["bla"].(*MyMod).running)
}

func TestShutdownModules(t *testing.T) {
	resetModules()
	initApp()
	Configure(true, "testtype", Standalone, map[string]string{}, viper.New())

	RegisterModule(&MyMod{}, "bla")
	startModules()

	shutdownModules()
	assert.Equal(t, false, modules["bla"].(*MyMod).running)
}
