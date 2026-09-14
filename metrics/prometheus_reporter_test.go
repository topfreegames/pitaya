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
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/topfreegames/pitaya/v2/config"
	"github.com/topfreegames/pitaya/v2/metrics/models"
)

func gatherFamily(t *testing.T, name string) *dto.MetricFamily {
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == name {
			return family
		}
	}
	return nil
}

func TestPrometheusReporterRegistersHistograms(t *testing.T) {
	cfg := config.MetricsConfig{
		Game:             "game",
		AdditionalLabels: map[string]string{},
		ConstLabels:      map[string]string{},
		Prometheus:       &config.PrometheusConfig{Enabled: true, Port: 0},
		Custom: models.CustomMetricsSpec{
			Histograms: []*models.Histogram{{
				Subsystem: "custom",
				Name:      "custom_histogram",
				Help:      "custom histogram",
				Buckets:   []float64{1, 10, 100},
			}},
		},
	}

	reporter, err := getPrometheusReporter("connector", cfg, &cfg.Custom)
	require.NoError(t, err)

	require.NoError(t, reporter.ReportHistogram("custom_histogram", nil, 5))
	require.NoError(t, reporter.ReportHistogram(ChannelCapacityHistogram, map[string]string{"channel": "ch"}, 42))

	for _, name := range []string{"pitaya_custom_custom_histogram", "pitaya_channel_channel_capacity_histogram"} {
		family := gatherFamily(t, name)
		require.NotNil(t, family, "%s is not registered", name)
		assert.Equal(t, dto.MetricType_HISTOGRAM, family.GetType())
		require.Len(t, family.GetMetric(), 1)
		assert.EqualValues(t, 1, family.GetMetric()[0].GetHistogram().GetSampleCount())
	}
}
