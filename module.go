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

import "github.com/topfreegames/pitaya/module"

var (
	// Modules are the modules that will be used by the app
	modules = make(map[string]module.Module)
)

func startModules() {
	log.Debug("initializing all modules")
	for name, mod := range modules {
		log.Debugf("initializing module: %s", name)
		if err := mod.Init(); err != nil {
			log.Errorf("error starting module %s, error: %s", name, err.Error())
		}
	}

	for name, mod := range modules {
		mod.AfterInit()
		log.Infof("module: %s successfully loaded", name)
	}
}

// shutdownModules shutdown the modules
func shutdownModules() {
	for _, mod := range modules {
		mod.BeforeShutdown()
	}
	for name, mod := range modules {
		log.Debugf("stopping module: %s", name)
		if err := mod.Shutdown(); err != nil {
			log.Warnf("error stopping module: %s", name)
		}
		log.Infof("module: %s stopped!", name)
	}
}
