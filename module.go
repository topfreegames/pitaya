// Copyright (c) TFG Co. All Rights Reserved.
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
	"fmt"

	"github.com/topfreegames/pitaya/v2/interfaces"
	"github.com/topfreegames/pitaya/v2/logger"
)

type moduleWrapper struct {
	module interfaces.Module
	name   string
}

// RegisterModule registers a module, by default it register after registered modules
func (app *App) RegisterModule(module interfaces.Module, name string) error {
	return app.RegisterModuleAfter(module, name)
}

// RegisterModuleAfter registers a module after all registered modules
func (app *App) RegisterModuleAfter(module interfaces.Module, name string) error {
	if err := app.alreadyRegistered(name); err != nil {
		return err
	}

	app.modulesMap[name] = module
	app.modulesArr = append(app.modulesArr, moduleWrapper{
		module: module,
		name:   name,
	})

	return nil
}

// RegisterModuleBefore registers a module before all registered modules
func (app *App) RegisterModuleBefore(module interfaces.Module, name string) error {
	if err := app.alreadyRegistered(name); err != nil {
		return err
	}

	app.modulesMap[name] = module
	app.modulesArr = append([]moduleWrapper{
		{
			module: module,
			name:   name,
		},
	}, app.modulesArr...)

	return nil
}

// GetModule gets a module with a name
func (app *App) GetModule(name string) (interfaces.Module, error) {
	if m, ok := app.modulesMap[name]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("module with name %s not found", name)
}

func (app *App) alreadyRegistered(name string) error {
	if _, ok := app.modulesMap[name]; ok {
		return fmt.Errorf("module with name %s already exists", name)
	}

	return nil
}

// startModules starts all modules in order
func (app *App) startModules() {
	logger.Log.Debug("initializing all modules")
	for _, modWrapper := range app.modulesArr {
		logger.Log.Debugf("initializing module: %s", modWrapper.name)
		if err := modWrapper.module.Init(); err != nil {
			logger.Log.Fatalf("error starting module %s, error: %s", modWrapper.name, err.Error())
		}
	}

	for _, modWrapper := range app.modulesArr {
		modWrapper.module.AfterInit()
		logger.Log.Infof("module: %s successfully loaded", modWrapper.name)
	}
}

// shutdownModules starts all modules in reverse order
func (app *App) shutdownModules() {
	for i := len(app.modulesArr) - 1; i >= 0; i-- {
		app.modulesArr[i].module.BeforeShutdown()
	}

	for i := len(app.modulesArr) - 1; i >= 0; i-- {
		name := app.modulesArr[i].name
		mod := app.modulesArr[i].module

		logger.Log.Debugf("stopping module: %s", name)
		if err := mod.Shutdown(); err != nil {
			logger.Log.Warnf("error stopping module: %s", name)
		}
		logger.Log.Infof("module: %s stopped!", name)
	}
}
