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

	"github.com/topfreegames/pitaya/v2/logger"
)

type (
	// HandlerTempl is a function that has the same signature as a handler and will
	// be called before or after handler methods
	HandlerTempl func(ctx context.Context, in interface{}) (out interface{}, err error)

	// AfterHandlerTempl is a function for the after handler, receives both the handler response
	// and the error returned
	AfterHandlerTempl func(ctx context.Context, out interface{}, err error) (interface{}, error)

	// Channel contains the functions to be called before the handler method is executed
	Channel struct {
		Handlers []HandlerTempl
	}

	// AfterChannel contains the functions to be called after the handler method is executed
	AfterChannel struct {
		Handlers []AfterHandlerTempl
	}

	// HandlerHooks contains before and after channels
	HandlerHooks struct {
		BeforeHandler *Channel
		AfterHandler  *AfterChannel
	}
)

// NewHandlerHooks ctor
func NewHandlerHooks() *HandlerHooks {
	return &HandlerHooks{
		BeforeHandler: NewChannel(),
		AfterHandler:  NewAfterChannel(),
	}
}

// NewChannel ctor
func NewChannel() *Channel {
	return &Channel{Handlers: []HandlerTempl{}}
}

// NewAfterChannel ctor
func NewAfterChannel() *AfterChannel {
	return &AfterChannel{Handlers: []AfterHandlerTempl{}}
}

// ExecuteBeforePipeline calls registered handlers
func (p *Channel) ExecuteBeforePipeline(ctx context.Context, data interface{}) (interface{}, error) {
	var err error
	res := data
	if len(p.Handlers) > 0 {
		for _, h := range p.Handlers {
			res, err = h(ctx, res)
			if err != nil {
				logger.Log.Debugf("pitaya/handler: broken pipeline: %s", err.Error())
				return res, err
			}
		}
	}
	return res, nil
}

// ExecuteAfterPipeline calls registered handlers
func (p *AfterChannel) ExecuteAfterPipeline(ctx context.Context, res interface{}, err error) (interface{}, error) {
	ret := res
	if len(p.Handlers) > 0 {
		for _, h := range p.Handlers {
			ret, err = h(ctx, ret, err)
		}
	}
	return ret, err
}

// PushFront should not be used after pitaya is running
func (p *Channel) PushFront(h HandlerTempl) {
	Handlers := make([]HandlerTempl, len(p.Handlers)+1)
	Handlers[0] = h
	copy(Handlers[1:], p.Handlers)
	p.Handlers = Handlers
}

// PushBack should not be used after pitaya is running
func (p *Channel) PushBack(h HandlerTempl) {
	p.Handlers = append(p.Handlers, h)
}

// Clear should not be used after pitaya is running
func (p *Channel) Clear() {
	p.Handlers = make([]HandlerTempl, 0)
}

// PushFront should not be used after pitaya is running
func (p *AfterChannel) PushFront(h AfterHandlerTempl) {
	Handlers := make([]AfterHandlerTempl, len(p.Handlers)+1)
	Handlers[0] = h
	copy(Handlers[1:], p.Handlers)
	p.Handlers = Handlers
}

// PushBack should not be used after pitaya is running
func (p *AfterChannel) PushBack(h AfterHandlerTempl) {
	p.Handlers = append(p.Handlers, h)
}

// Clear should not be used after pitaya is running
func (p *AfterChannel) Clear() {
	p.Handlers = make([]AfterHandlerTempl, 0)
}
