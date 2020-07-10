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

type MyMod struct {
	component.Base
	running bool
	name    string
}

var modulesOrder []string

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
	b := &MyMod{}

	config := viper.New()
	app := NewDefaultApp(true, "testtype", Cluster, map[string]string{}, config)

	err := app.RegisterModule(b, "mod")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(app.modulesMap))
	assert.Equal(t, b, app.modulesMap["mod"])
	assert.Equal(t, 1, len(app.modulesArr))
	assert.Equal(t, "mod", app.modulesArr[0].name)
	assert.Equal(t, b, app.modulesArr[0].module)
	err = app.RegisterModule(b, "mod")
	assert.Error(t, err)
}

func TestGetModule(t *testing.T) {
	b := &MyMod{}

	config := viper.New()
	app := NewDefaultApp(true, "testtype", Cluster, map[string]string{}, config)

	app.RegisterModule(b, "mod")
	m, err := app.GetModule("mod")
	assert.NoError(t, err)
	assert.Equal(t, b, m)

	m, err = app.GetModule("mmm")
	assert.Error(t, err)
}

func TestStartupModules(t *testing.T) {
	modulesOrder = []string{}
	app := NewDefaultApp(true, "testtype", Standalone, map[string]string{}, viper.New())

	err := app.RegisterModule(&MyMod{name: "mod1"}, "mod1")
	assert.NoError(t, err)
	err = app.RegisterModuleBefore(&MyMod{name: "mod2"}, "mod2")
	assert.NoError(t, err)
	err = app.RegisterModuleBefore(&MyMod{name: "mod3"}, "mod3")
	assert.NoError(t, err)
	err = app.RegisterModuleAfter(&MyMod{name: "mod4"}, "mod4")
	assert.NoError(t, err)

	app.startModules()
	assert.Equal(t, true, app.modulesMap["mod1"].(*MyMod).running)
	assert.Equal(t, true, app.modulesMap["mod2"].(*MyMod).running)
	assert.Equal(t, true, app.modulesMap["mod3"].(*MyMod).running)
	assert.Equal(t, true, app.modulesMap["mod4"].(*MyMod).running)
	assert.Equal(t, []string{"mod3", "mod2", "mod1", "mod4"}, modulesOrder)
}

func TestShutdownModules(t *testing.T) {
	modulesOrder = []string{}
	app := NewDefaultApp(true, "testtype", Standalone, map[string]string{}, viper.New())

	err := app.RegisterModule(&MyMod{name: "mod1"}, "mod1")
	assert.NoError(t, err)
	err = app.RegisterModuleBefore(&MyMod{name: "mod2"}, "mod2")
	assert.NoError(t, err)
	err = app.RegisterModuleBefore(&MyMod{name: "mod3"}, "mod3")
	assert.NoError(t, err)
	err = app.RegisterModuleAfter(&MyMod{name: "mod4"}, "mod4")
	assert.NoError(t, err)

	app.startModules()

	modulesOrder = []string{}
	app.shutdownModules()
	assert.Equal(t, false, app.modulesMap["mod1"].(*MyMod).running)
	assert.Equal(t, false, app.modulesMap["mod2"].(*MyMod).running)
	assert.Equal(t, false, app.modulesMap["mod3"].(*MyMod).running)
	assert.Equal(t, false, app.modulesMap["mod4"].(*MyMod).running)
	assert.Equal(t, []string{"mod4", "mod1", "mod2", "mod3"}, modulesOrder)
}
