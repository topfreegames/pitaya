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
	"container/list"
	"time"

	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/metrics"
)

// RateLimiter wraps net.Conn by applying rate limiting and return empty
// if exceeded. It uses the leaky bucket
// algorithm (https://en.wikipedia.org/wiki/Leaky_bucket).
// Here, "limit" is the number of requests it accepts during an "interval" duration.
// After making a request, a slot is occupied and only freed after "interval"
// duration. If a new request comes when no slots are available, the buffer from
// Read is droped and ignored by pitaya.
// On the client side, this will yield a timeout error and the client must
// be prepared to handle it.
type RateLimiter struct {
	acceptor.PlayerConn
	limit        int
	interval     time.Duration
	times        list.List
	forceDisable bool
}

// NewRateLimiter returns an initialized *RateLimiting
func NewRateLimiter(
	conn acceptor.PlayerConn,
	limit int,
	interval time.Duration,
	forceDisable bool,
) *RateLimiter {
	r := &RateLimiter{
		PlayerConn:   conn,
		limit:        limit,
		interval:     interval,
		forceDisable: forceDisable,
	}

	r.times.Init()

	return r
}

// GetNextMessage gets the next message in the connection
func (r *RateLimiter) GetNextMessage() (msg []byte, err error) {
	if r.forceDisable {
		return r.PlayerConn.GetNextMessage()
	}

	for {
		msg, err := r.PlayerConn.GetNextMessage()
		if err != nil {
			return nil, err
		}

		now := time.Now()
		if r.shouldRateLimit(now) {
			logger.Log.Errorf("Data=%s, Error=%s", msg, constants.ErrRateLimitExceeded)
			metrics.ReportExceededRateLimiting(pitaya.GetMetricsReporters())
			continue
		}

		return msg, err
	}
}

// shouldRateLimit saves the now as time taken or returns an error if
// in the limit of rate limiting
func (r *RateLimiter) shouldRateLimit(now time.Time) bool {
	if r.times.Len() < r.limit {
		r.times.PushBack(now)
		return false
	}

	front := r.times.Front()
	if diff := now.Sub(front.Value.(time.Time)); diff < r.interval {
		return true
	}

	front.Value = now
	r.times.MoveToBack(front)
	return false
}
