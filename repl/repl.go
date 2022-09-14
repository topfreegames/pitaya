/*
Copyright © 2021 Wildlife Studios

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package repl

import (
	"github.com/topfreegames/pitaya/v2/pkg/client"
	"sync"

	"github.com/topfreegames/pitaya/v2/pkg/session"
)

var (
	pClient        client.PitayaClient
	disconnectedCh chan bool
	docsString     string
	fileName       string
	pushInfo       map[string]string
	wait           sync.WaitGroup
	prettyJSON     bool
	handshake      *session.HandshakeData
)

func Start(docs, filename string, prettyJSON bool) {
	docsString = docs
	fileName = filename
	handshake = &session.HandshakeData{
		Sys: session.HandshakeClientData{
			Platform:    "repl",
			LibVersion:  "1.3.1",
			BuildNumber: "20",
			Version:     "1.0.0",
		},
		User: map[string]interface{}{
			"client": "repl",
		},
	}

	switch {
	case fileName != "":
		executeFromFile(fileName)
	default:
		repl()
	}
}
