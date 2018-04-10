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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/pipeline"
	"github.com/topfreegames/pitaya/session"
)

func resetPipelines() {
	pipeline.BeforeHandler.Handlers = make([]pipeline.Handler, 0)
	pipeline.AfterHandler.Handlers = make([]pipeline.Handler, 0)
}

var myHandler = func(s *session.Session, in []byte) ([]byte, error) {
	return []byte("test"), nil
}

func TestBeforeHandler(t *testing.T) {
	resetPipelines()
	BeforeHandler(myHandler)
	r, err := pipeline.BeforeHandler.Handlers[0](nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), r)
}

func TestAfterHandler(t *testing.T) {
	resetPipelines()
	AfterHandler(myHandler)
	r, err := pipeline.AfterHandler.Handlers[0](nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), r)
}
