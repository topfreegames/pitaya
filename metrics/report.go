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
	"time"

	"github.com/topfreegames/pitaya/constants"
	pcontext "github.com/topfreegames/pitaya/context"
)

// ReportTimingFromCtx reports the latency from the context
func ReportTimingFromCtx(ctx context.Context, reporters []Reporter, typ string, errored bool) {
	status := "ok"
	if errored {
		status = "failed"
	}
	if len(reporters) > 0 {
		startTime := pcontext.GetFromPropagateCtx(ctx, constants.StartTimeKey)
		route := pcontext.GetFromPropagateCtx(ctx, constants.RouteKey)
		elapsed := time.Since(time.Unix(0, startTime.(int64)))
		tags := map[string]string{
			"route":  route.(string),
			"status": status,
			"type":   typ,
		}
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
		tags := map[string]string{
			"type": typ,
		}
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
