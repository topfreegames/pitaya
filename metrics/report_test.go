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
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/constants"
	pcontext "github.com/topfreegames/pitaya/context"
	"github.com/topfreegames/pitaya/metrics/mocks"
)

func TestReportTimingFromCtx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockMetricsReporter := mocks.NewMockReporter(ctrl)

	originalTs := time.Now().UnixNano()
	expectedRoute := uuid.New().String()
	expectedType := uuid.New().String()
	expectedErrored := true
	ctx := pcontext.AddToPropagateCtx(context.Background(), constants.StartTimeKey, originalTs)
	ctx = pcontext.AddToPropagateCtx(ctx, constants.RouteKey, expectedRoute)

	time.Sleep(200 * time.Millisecond) // to test duration report
	mockMetricsReporter.EXPECT().ReportLatency(gomock.Any(), expectedRoute, expectedType, expectedErrored).Do(func(duration time.Duration, r, tt string, e bool) {
		assert.InDelta(t, duration.Nanoseconds(), time.Now().UnixNano()-originalTs, 10e6)
	})

	ReportTimingFromCtx(ctx, mockMetricsReporter, expectedType, expectedErrored)
}
