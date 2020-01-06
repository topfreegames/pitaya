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

package metrics

import (
	"context"
	"runtime"
	"time"

	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/errors"

	pcontext "github.com/topfreegames/pitaya/context"
)

// ReportTimingFromCtx reports the latency from the context
func ReportTimingFromCtx(ctx context.Context, reporters []Reporter, typ string, err error) {
	if ctx == nil {
		return
	}
	code := errors.CodeFromError(err)
	status := "ok"
	if err != nil {
		status = "failed"
	}
	if len(reporters) > 0 {
		startTime := pcontext.GetFromPropagateCtx(ctx, constants.StartTimeKey)
		route := pcontext.GetFromPropagateCtx(ctx, constants.RouteKey)
		elapsed := time.Since(time.Unix(0, startTime.(int64)))
		tags := getTags(ctx, map[string]string{
			"route":  route.(string),
			"status": status,
			"type":   typ,
			"code":   code,
		})
		for _, r := range reporters {
			r.ReportSummary(ResponseTime, tags, float64(elapsed.Nanoseconds()))
		}
	}
}

// ReportMessageProcessDelayFromCtx reports the delay to process the messages
func ReportMessageProcessDelayFromCtx(ctx context.Context, reporters []Reporter, typ string) {
	if len(reporters) > 0 {
		startTime := pcontext.GetFromPropagateCtx(ctx, constants.StartTimeKey)
		elapsed := time.Since(time.Unix(0, startTime.(int64)))
		route := pcontext.GetFromPropagateCtx(ctx, constants.RouteKey)
		tags := getTags(ctx, map[string]string{
			"route": route.(string),
			"type":  typ,
		})
		for _, r := range reporters {
			r.ReportSummary(ProcessDelay, tags, float64(elapsed.Nanoseconds()))
		}
	}
}

// ReportNumberOfConnectedClients reports the number of connected clients
func ReportNumberOfConnectedClients(reporters []Reporter, number int64) {
	for _, r := range reporters {
		r.ReportGauge(ConnectedClients, map[string]string{}, float64(number))
	}
}

// ReportSysMetrics reports sys metrics
func ReportSysMetrics(reporters []Reporter, period time.Duration) {
	for {
		for _, r := range reporters {
			num := runtime.NumGoroutine()
			m := &runtime.MemStats{}
			runtime.ReadMemStats(m)

			r.ReportGauge(Goroutines, map[string]string{}, float64(num))
			r.ReportGauge(HeapSize, map[string]string{}, float64(m.Alloc))
			r.ReportGauge(HeapObjects, map[string]string{}, float64(m.HeapObjects))
		}

		time.Sleep(period)
	}
}

// ReportExceededRateLimiting reports the number of requests made
// after exceeded rate limiting in a connection
func ReportExceededRateLimiting(reporters []Reporter) {
	for _, r := range reporters {
		r.ReportCount(ExceededRateLimiting, map[string]string{}, 1)
	}
}

func tagsFromContext(ctx context.Context) map[string]string {
	val := pcontext.GetFromPropagateCtx(ctx, constants.MetricTagsKey)
	if val == nil {
		return map[string]string{}
	}

	tags, ok := val.(map[string]string)
	if !ok {
		return map[string]string{}
	}

	return tags
}

func getTags(ctx context.Context, tags map[string]string) map[string]string {
	for k, v := range tagsFromContext(ctx) {
		tags[k] = v
	}

	return tags
}
