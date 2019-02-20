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
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/constants"
	pcontext "github.com/topfreegames/pitaya/context"
	e "github.com/topfreegames/pitaya/errors"
	"github.com/topfreegames/pitaya/metrics/mocks"
)

func TestReportTimingFromCtx(t *testing.T) {
	t.Run("test-duration", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockMetricsReporter := mocks.NewMockReporter(ctrl)

		originalTs := time.Now().UnixNano()
		expectedRoute := uuid.New().String()
		expectedType := uuid.New().String()
		expectedErr := errors.New(uuid.New().String())
		ctx := pcontext.AddToPropagateCtx(context.Background(), constants.StartTimeKey, originalTs)
		ctx = pcontext.AddToPropagateCtx(ctx, constants.RouteKey, expectedRoute)

		time.Sleep(200 * time.Millisecond) // to test duration report
		mockMetricsReporter.EXPECT().ReportSummary(ResponseTime, gomock.Any(), gomock.Any()).Do(
			func(metric string, tags map[string]string, duration float64) {
				assert.InDelta(t, duration, time.Now().UnixNano()-originalTs, 10e6)
			},
		)

		ReportTimingFromCtx(ctx, []Reporter{mockMetricsReporter}, expectedType, expectedErr)
	})

	t.Run("test-tags", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockMetricsReporter := mocks.NewMockReporter(ctrl)

		originalTs := time.Now().UnixNano()
		expectedRoute := uuid.New().String()
		expectedType := uuid.New().String()
		var expectedErr error
		ctx := pcontext.AddToPropagateCtx(context.Background(), constants.StartTimeKey, originalTs)
		ctx = pcontext.AddToPropagateCtx(ctx, constants.RouteKey, expectedRoute)
		ctx = pcontext.AddToPropagateCtx(ctx, constants.MetricTagsKey, map[string]string{
			"key": "value",
		})

		expectedTags := map[string]string{
			"route":  expectedRoute,
			"status": "ok",
			"type":   expectedType,
			"key":    "value",
			"code":   "",
		}

		mockMetricsReporter.EXPECT().ReportSummary(ResponseTime, expectedTags, gomock.Any())

		ReportTimingFromCtx(ctx, []Reporter{mockMetricsReporter}, expectedType, expectedErr)
	})

	t.Run("test-tags-not-correct-type", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockMetricsReporter := mocks.NewMockReporter(ctrl)

		originalTs := time.Now().UnixNano()
		expectedRoute := uuid.New().String()
		expectedType := uuid.New().String()
		var expectedErr error
		ctx := pcontext.AddToPropagateCtx(context.Background(), constants.StartTimeKey, originalTs)
		ctx = pcontext.AddToPropagateCtx(ctx, constants.RouteKey, expectedRoute)
		ctx = pcontext.AddToPropagateCtx(ctx, constants.MetricTagsKey, "not-map")

		expectedTags := map[string]string{
			"route":  expectedRoute,
			"status": "ok",
			"type":   expectedType,
			"code":   "",
		}

		mockMetricsReporter.EXPECT().ReportSummary(ResponseTime, expectedTags, gomock.Any())

		ReportTimingFromCtx(ctx, []Reporter{mockMetricsReporter}, expectedType, expectedErr)
	})

	t.Run("test-failed-route-with-pitaya-error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockMetricsReporter := mocks.NewMockReporter(ctrl)

		originalTs := time.Now().UnixNano()
		expectedRoute := uuid.New().String()
		expectedType := uuid.New().String()
		code := "GAME-404"
		expectedErr := e.NewError(errors.New("error"), code)
		ctx := pcontext.AddToPropagateCtx(context.Background(), constants.StartTimeKey, originalTs)
		ctx = pcontext.AddToPropagateCtx(ctx, constants.RouteKey, expectedRoute)

		mockMetricsReporter.EXPECT().ReportSummary(ResponseTime, map[string]string{
			"route":  expectedRoute,
			"status": "failed",
			"type":   expectedType,
			"code":   code,
		}, gomock.Any())

		ReportTimingFromCtx(ctx, []Reporter{mockMetricsReporter}, expectedType, expectedErr)
	})

	t.Run("test-failed-route", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockMetricsReporter := mocks.NewMockReporter(ctrl)

		originalTs := time.Now().UnixNano()
		expectedRoute := uuid.New().String()
		expectedType := uuid.New().String()
		expectedErr := errors.New("error")
		ctx := pcontext.AddToPropagateCtx(context.Background(), constants.StartTimeKey, originalTs)
		ctx = pcontext.AddToPropagateCtx(ctx, constants.RouteKey, expectedRoute)

		mockMetricsReporter.EXPECT().ReportSummary(ResponseTime, map[string]string{
			"route":  expectedRoute,
			"status": "failed",
			"type":   expectedType,
			"code":   e.ErrUnknownCode,
		}, gomock.Any())

		ReportTimingFromCtx(ctx, []Reporter{mockMetricsReporter}, expectedType, expectedErr)
	})
}

func TestReportMessageProcessDelayFromCtx(t *testing.T) {
	t.Run("test-tags", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockMetricsReporter := mocks.NewMockReporter(ctrl)

		originalTs := time.Now().UnixNano()
		expectedRoute := uuid.New().String()
		expectedType := uuid.New().String()
		ctx := pcontext.AddToPropagateCtx(context.Background(), constants.StartTimeKey, originalTs)
		ctx = pcontext.AddToPropagateCtx(ctx, constants.RouteKey, expectedRoute)
		ctx = pcontext.AddToPropagateCtx(ctx, constants.MetricTagsKey, map[string]string{
			"key": "value",
		})

		expectedTags := map[string]string{
			"route": expectedRoute,
			"type":  expectedType,
			"key":   "value",
		}

		mockMetricsReporter.EXPECT().ReportSummary(ProcessDelay, expectedTags, gomock.Any())

		ReportMessageProcessDelayFromCtx(ctx, []Reporter{mockMetricsReporter}, expectedType)
	})
}
