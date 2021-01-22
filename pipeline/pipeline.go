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

import "context"

var (
	// BeforeHandler contains the functions to be called before the handler method is executed
	BeforeHandler = &pipelineChannel{}
	// AfterHandler contains the functions to be called after the handler method is executed
	AfterHandler = &pipelineAfterChannel{}
)

type (
	// HandlerTempl is a function that has the same signature as a handler and will
	// be called before or after handler methods
	HandlerTempl func(ctx context.Context, in interface{}) (c context.Context, out interface{}, err error)

	// AfterHandlerTempl is a function for the after handler, receives both the handler response
	// and the error returned
	AfterHandlerTempl func(ctx context.Context, out interface{}, err error) (interface{}, error)

	pipelineChannel struct {
		Handlers []HandlerTempl
	}

	pipelineAfterChannel struct {
		Handlers []AfterHandlerTempl
	}
)

// PushFront should not be used after pitaya is running
func (p *pipelineChannel) PushFront(h HandlerTempl) {
	Handlers := make([]HandlerTempl, len(p.Handlers)+1)
	Handlers[0] = h
	copy(Handlers[1:], p.Handlers)
	p.Handlers = Handlers
}

// PushBack should not be used after pitaya is running
func (p *pipelineChannel) PushBack(h HandlerTempl) {
	p.Handlers = append(p.Handlers, h)
}

// Clear should not be used after pitaya is running
func (p *pipelineChannel) Clear() {
	p.Handlers = make([]HandlerTempl, 0)
}

// PushFront should not be used after pitaya is running
func (p *pipelineAfterChannel) PushFront(h AfterHandlerTempl) {
	Handlers := make([]AfterHandlerTempl, len(p.Handlers)+1)
	Handlers[0] = h
	copy(Handlers[1:], p.Handlers)
	p.Handlers = Handlers
}

// PushBack should not be used after pitaya is running
func (p *pipelineAfterChannel) PushBack(h AfterHandlerTempl) {
	p.Handlers = append(p.Handlers, h)
}

// Clear should not be used after pitaya is running
func (p *pipelineAfterChannel) Clear() {
	p.Handlers = make([]AfterHandlerTempl, 0)
}
