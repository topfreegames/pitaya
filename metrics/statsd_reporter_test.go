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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/config"
	metricsmocks "github.com/topfreegames/pitaya/metrics/mocks"
)

func TestNewStatsdReporter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := metricsmocks.NewMockClient(ctrl)

	cfg := config.NewConfig()
	sr, err := NewStatsdReporter(cfg, "svType", mockClient)
	assert.NoError(t, err)
	assert.Equal(t, mockClient, sr.client)
	assert.Equal(t, float64(cfg.GetInt("pitaya.metrics.statsd.rate")), sr.rate)
	assert.Equal(t, "svType", sr.serverType)
	assert.NotEmpty(t, sr.hostname)
}

func TestReportLatency(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := metricsmocks.NewMockClient(ctrl)

	cfg := config.NewConfig()
	sr, err := NewStatsdReporter(cfg, "svType", mockClient)
	assert.NoError(t, err)

	expectedDuration, err := time.ParseDuration("200ms")
	assert.NoError(t, err)
	expectedRoute := uuid.New().String()
	expectedType := uuid.New().String()
	expectedErrored := true

	mockClient.EXPECT().Timing("response_time_ms", expectedDuration, gomock.Any(), sr.rate).Do(func(n string, d time.Duration, tags []string, r float64) {
		assert.Contains(t, tags, fmt.Sprintf("route:%s", expectedRoute))
		assert.Contains(t, tags, fmt.Sprintf("type:%s", expectedType))
		assert.Contains(t, tags, fmt.Sprintf("error:%t", expectedErrored))
		assert.Contains(t, tags, fmt.Sprintf("serverType:%s", sr.serverType))
		assert.Contains(t, tags, fmt.Sprintf("hostname:%s", sr.hostname))
	})

	err = sr.ReportLatency(expectedDuration, expectedRoute, expectedType, expectedErrored)
	assert.NoError(t, err)
}

func TestReportLatencyError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := metricsmocks.NewMockClient(ctrl)

	cfg := config.NewConfig()
	sr, err := NewStatsdReporter(cfg, "svType", mockClient)
	assert.NoError(t, err)

	expectedError := errors.New("some error")
	mockClient.EXPECT().Timing("response_time_ms", gomock.Any(), gomock.Any(), sr.rate).Return(expectedError)

	err = sr.ReportLatency(123, "", "", false)
	assert.Equal(t, expectedError, err)
}

func TestReportCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := metricsmocks.NewMockClient(ctrl)

	cfg := config.NewConfig()
	sr, err := NewStatsdReporter(cfg, "svType", mockClient)
	assert.NoError(t, err)

	expectedCount := 123
	expectedMetric := uuid.New().String()
	customTags := []string{
		fmt.Sprintf("tag1:%s", uuid.New().String()),
		fmt.Sprintf("tag2:%s", uuid.New().String()),
	}
	mockClient.EXPECT().Gauge(expectedMetric, float64(expectedCount), gomock.Any(), sr.rate).Do(func(n string, v float64, tags []string, r float64) {
		for _, tag := range customTags {
			assert.Contains(t, tags, tag)
		}
		assert.Contains(t, tags, fmt.Sprintf("serverType:%s", sr.serverType))
		assert.Contains(t, tags, fmt.Sprintf("hostname:%s", sr.hostname))
	})

	err = sr.ReportCount(expectedCount, expectedMetric, customTags...)
	assert.NoError(t, err)
}

func TestReportCountError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := metricsmocks.NewMockClient(ctrl)

	cfg := config.NewConfig()
	sr, err := NewStatsdReporter(cfg, "svType", mockClient)
	assert.NoError(t, err)

	expectedError := errors.New("some error")
	mockClient.EXPECT().Gauge(gomock.Any(), gomock.Any(), gomock.Any(), sr.rate).Return(expectedError)

	err = sr.ReportCount(123, "")
	assert.Equal(t, expectedError, err)
}
