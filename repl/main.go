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

package repl

import (
	"sync"

	"github.com/topfreegames/pitaya/v3/pkg/client"
	"github.com/topfreegames/pitaya/v3/pkg/session"
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
	prettyJSON = prettyJSON
	handshake = &session.HandshakeData{
		Sys: session.HandshakeClientData{
			Platform:    "repl",
			LibVersion:  "0.3.5-release",
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
