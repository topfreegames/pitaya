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

package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	handler1 = func(ctx context.Context, in interface{}) (context.Context, interface{}, error) {
		return ctx, in, errors.New("ohno")
	}
	handler2 = func(ctx context.Context, in interface{}) (context.Context, interface{}, error) {
		return ctx, nil, nil
	}
	p = &pipelineChannel{}
)

func TestPushFront(t *testing.T) {
	p.PushFront(handler1)
	p.PushFront(handler2)
	defer p.Clear()

	_, _, err := p.Handlers[0](nil, nil)
	assert.Nil(t, nil, err)
}

func TestPushBack(t *testing.T) {
	p.PushFront(handler1)
	p.PushBack(handler2)
	defer p.Clear()

	_, _, err := p.Handlers[0](nil, nil)
	assert.EqualError(t, errors.New("ohno"), err.Error())
}

func TestClear(t *testing.T) {
	p.PushFront(handler1)
	p.PushBack(handler2)
	assert.Len(t, p.Handlers, 2)
	p.Clear()
	assert.Len(t, p.Handlers, 0)
}
