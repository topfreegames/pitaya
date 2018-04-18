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

	"github.com/topfreegames/pitaya/interfaces"
	"github.com/topfreegames/pitaya/logger"
)

var modules = make(map[string]interfaces.Module)

// RegisterModule registers a module
func RegisterModule(module interfaces.Module, name string) error {
	if _, ok := modules[name]; ok {
		return fmt.Errorf("module with name %s already exists", name)
	}
	modules[name] = module
	return nil
}

// GetModule gets a module with a name
func GetModule(name string) (interfaces.Module, error) {
	if m, ok := modules[name]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("module with name %s not found", name)
}

// StartModules starts all modules
func startModules() {
	logger.Log.Debug("initializing all modules")
	for name, mod := range modules {
		logger.Log.Debugf("initializing module: %s", name)
		if err := mod.Init(); err != nil {
			logger.Log.Fatalf("error starting module %s, error: %s", name, err.Error())
		}
	}

	for name, mod := range modules {
		mod.AfterInit()
		logger.Log.Infof("module: %s successfully loaded", name)
	}
}

func shutdownModules() {
	for _, mod := range modules {
		mod.BeforeShutdown()
	}
	for name, mod := range modules {
		logger.Log.Debugf("stopping module: %s", name)
		if err := mod.Shutdown(); err != nil {
			logger.Log.Warnf("error stopping module: %s", name)
		}
		logger.Log.Infof("module: %s stopped!", name)
	}
}
