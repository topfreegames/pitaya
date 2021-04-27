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

package acceptorwrapper

import (
	"github.com/topfreegames/pitaya/v2/acceptor"
	"github.com/topfreegames/pitaya/v2/config"
	"github.com/topfreegames/pitaya/v2/metrics"
)

// RateLimitingWrapper rate limits for each connection
// received
type RateLimitingWrapper struct {
	BaseWrapper
}

// NewRateLimitingWrapper returns an instance of *RateLimitingWrapper
func NewRateLimitingWrapper(reporters []metrics.Reporter, c config.RateLimitingConfig) *RateLimitingWrapper {
	r := &RateLimitingWrapper{}

	r.BaseWrapper = NewBaseWrapper(func(conn acceptor.PlayerConn) acceptor.PlayerConn {
		return NewRateLimiter(reporters, conn, c.Limit, c.Interval, c.ForceDisable)
	})

	return r
}

// Wrap saves acceptor as an attribute
func (r *RateLimitingWrapper) Wrap(a acceptor.Acceptor) acceptor.Acceptor {
	r.Acceptor = a
	return r
}
