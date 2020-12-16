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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/pipeline"
)

func resetPipelines() {
	pipeline.BeforeHandler.Handlers = make([]pipeline.HandlerTempl, 0)
	pipeline.AfterHandler.Handlers = make([]pipeline.AfterHandlerTempl, 0)
}

var myHandler = func(ctx context.Context, in interface{}) (context.Context, interface{}, error) {
	ctx = context.WithValue(ctx, "traceID", "123456")
	return ctx, []byte("test"), nil
}

var myAfterHandler = func(ctx context.Context, out interface{}, err error) (interface{}, error) {
	return []byte("test"), nil
}

func TestBeforeHandler(t *testing.T) {
	resetPipelines()
	BeforeHandler(myHandler)
	ctx, r, err := pipeline.BeforeHandler.Handlers[0](context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), r)
	assert.Equal(t, "123456", ctx.Value("traceID").(string))
}

func TestAfterHandler(t *testing.T) {
	resetPipelines()
	AfterHandler(myAfterHandler)
	r, err := pipeline.AfterHandler.Handlers[0](nil, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), r)
}
