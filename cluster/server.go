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

package cluster

import (
	"encoding/json"
	"os"

	"github.com/topfreegames/pitaya/logger"
)

// Server struct
type Server struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Metadata map[string]string `json:"metadata"`
	Frontend bool              `json:"frontend"`
	Hostname string            `json:"hostname"`
}

// NewServer ctor
func NewServer(id, serverType string, frontend bool, metadata ...map[string]string) *Server {
	d := make(map[string]string)
	h, err := os.Hostname()
	if err != nil {
		logger.Log.Errorf("failed to get hostname: %s", err.Error())
	}
	if len(metadata) > 0 {
		d = metadata[0]
	}
	return &Server{
		ID:       id,
		Type:     serverType,
		Metadata: d,
		Frontend: frontend,
		Hostname: h,
	}
}

// AsJSONString returns the server as a json string
func (s *Server) AsJSONString() string {
	str, err := json.Marshal(s)
	if err != nil {
		logger.Log.Errorf("error getting server as json: %s", err.Error())
		return ""
	}
	return string(str)
}
