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
	name    string
}

var modulesOrder []string

func resetModules() {
	modulesMap = make(map[string]interfaces.Module)
	modulesArr = []moduleWrapper{}
	modulesOrder = []string{}
}

func (m *MyMod) Init() error {
	m.running = true
	modulesOrder = append(modulesOrder, m.name)
	return nil
}

func (m *MyMod) Shutdown() error {
	m.running = false
	modulesOrder = append(modulesOrder, m.name)
	return nil
}

func TestRegisterModule(t *testing.T) {
	resetModules()
	b := &MyMod{}
	err := RegisterModule(b, "mod")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(modulesMap))
	assert.Equal(t, b, modulesMap["mod"])
	assert.Equal(t, 1, len(modulesArr))
	assert.Equal(t, "mod", modulesArr[0].name)
	assert.Equal(t, b, modulesArr[0].module)
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

	err := RegisterModule(&MyMod{name: "mod1"}, "mod1")
	assert.NoError(t, err)
	err = RegisterModuleBefore(&MyMod{name: "mod2"}, "mod2")
	assert.NoError(t, err)
	err = RegisterModuleBefore(&MyMod{name: "mod3"}, "mod3")
	assert.NoError(t, err)
	err = RegisterModuleAfter(&MyMod{name: "mod4"}, "mod4")
	assert.NoError(t, err)

	startModules()
	assert.Equal(t, true, modulesMap["mod1"].(*MyMod).running)
	assert.Equal(t, true, modulesMap["mod2"].(*MyMod).running)
	assert.Equal(t, true, modulesMap["mod3"].(*MyMod).running)
	assert.Equal(t, true, modulesMap["mod4"].(*MyMod).running)
	assert.Equal(t, []string{"mod3", "mod2", "mod1", "mod4"}, modulesOrder)
}

func TestShutdownModules(t *testing.T) {
	resetModules()
	initApp()
	Configure(true, "testtype", Standalone, map[string]string{}, viper.New())

	err := RegisterModule(&MyMod{name: "mod1"}, "mod1")
	assert.NoError(t, err)
	err = RegisterModuleBefore(&MyMod{name: "mod2"}, "mod2")
	assert.NoError(t, err)
	err = RegisterModuleBefore(&MyMod{name: "mod3"}, "mod3")
	assert.NoError(t, err)
	err = RegisterModuleAfter(&MyMod{name: "mod4"}, "mod4")
	assert.NoError(t, err)

	startModules()

	modulesOrder = []string{}
	shutdownModules()
	assert.Equal(t, false, modulesMap["mod1"].(*MyMod).running)
	assert.Equal(t, false, modulesMap["mod2"].(*MyMod).running)
	assert.Equal(t, false, modulesMap["mod3"].(*MyMod).running)
	assert.Equal(t, false, modulesMap["mod4"].(*MyMod).running)
	assert.Equal(t, []string{"mod4", "mod1", "mod2", "mod3"}, modulesOrder)
}
