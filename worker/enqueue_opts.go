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

package worker

import "github.com/topfreegames/pitaya/config"

// EnqueueOpts has retry options for worker
type EnqueueOpts struct {
	RetryEnabled      bool
	MaxRetries        int
	ExponentialFactor int
	MinDelayToRetry   int
	MaxDelayToRetry   int
	MaxRandom         int
}

// NewEnqueueOpts reads from config to build *EnqueueOpts
func NewEnqueueOpts(config *config.Config) *EnqueueOpts {
	return &EnqueueOpts{
		RetryEnabled:      config.GetBool("pitaya.worker.retry.enabled"),
		MaxRetries:        config.GetInt("pitaya.worker.retry.max"),
		ExponentialFactor: config.GetInt("pitaya.worker.retry.exponential"),
		MinDelayToRetry:   config.GetInt("pitaya.worker.retry.minDelay"),
		MaxDelayToRetry:   config.GetInt("pitaya.worker.retry.maxDelay"),
		MaxRandom:         config.GetInt("pitaya.worker.retry.maxRandom"),
	}
}
