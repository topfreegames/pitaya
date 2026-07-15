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
	"sort"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildObjectives(t *testing.T) {
	t.Parallel()

	t.Run("nil-config-falls-back-to-default", func(t *testing.T) {
		objectives, err := buildObjectives(nil)
		require.NoError(t, err)
		assert.Equal(t, map[float64]float64{0.7: 0.02, 0.95: 0.005, 0.99: 0.001}, objectives)
	})

	t.Run("custom-objectives-are-used-verbatim", func(t *testing.T) {
		objectives, err := buildObjectives(map[string]float64{"0.95": 0.005, "0.99": 0.001})
		require.NoError(t, err)
		assert.Equal(t, map[float64]float64{0.95: 0.005, 0.99: 0.001}, objectives)
	})

	t.Run("explicit-empty-map-yields-no-quantiles", func(t *testing.T) {
		objectives, err := buildObjectives(map[string]float64{})
		require.NoError(t, err)
		assert.Empty(t, objectives)
		assert.NotNil(t, objectives)
	})

	t.Run("invalid-quantile-key-errors", func(t *testing.T) {
		_, err := buildObjectives(map[string]float64{"not-a-float": 0.01})
		assert.Error(t, err)
	})
}

// quantilesFromSummary registers a summary built exactly as registerMetrics
// builds the built-in ones, observes a value, scrapes it, and returns the sorted
// list of quantiles emitted plus whether _sum/_count were present.
func quantilesFromSummary(t *testing.T, objectives map[float64]float64) (quantiles []float64, hasSum, hasCount bool) {
	t.Helper()

	registry := prometheus.NewRegistry()
	summary := newSummaryVec(
		"handler",
		ResponseTime,
		"the time to process a msg in nanoseconds",
		objectives,
		map[string]string{},
		[]string{"route"},
	)
	require.NoError(t, registry.Register(summary))

	summary.With(map[string]string{"route": "test.route"}).Observe(42)

	families, err := registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)

	for _, m := range families[0].GetMetric() {
		s := m.GetSummary()
		for _, q := range s.GetQuantile() {
			quantiles = append(quantiles, q.GetQuantile())
		}
		hasSum = s.SampleSum != nil
		hasCount = s.SampleCount != nil
	}
	sort.Float64s(quantiles)
	return quantiles, hasSum, hasCount
}

func TestBuiltinSummaryObjectivesScrape(t *testing.T) {
	t.Parallel()

	t.Run("default-emits-standard-quantiles", func(t *testing.T) {
		objectives, err := buildObjectives(nil)
		require.NoError(t, err)

		quantiles, hasSum, hasCount := quantilesFromSummary(t, objectives)
		assert.Equal(t, []float64{0.7, 0.95, 0.99}, quantiles)
		assert.True(t, hasSum)
		assert.True(t, hasCount)
	})

	t.Run("custom-objectives-drop-unconfigured-quantiles", func(t *testing.T) {
		objectives, err := buildObjectives(map[string]float64{"0.95": 0.005, "0.99": 0.001})
		require.NoError(t, err)

		quantiles, hasSum, hasCount := quantilesFromSummary(t, objectives)
		assert.Equal(t, []float64{0.95, 0.99}, quantiles)
		assert.True(t, hasSum)
		assert.True(t, hasCount)
	})

	t.Run("empty-objectives-emit-only-sum-and-count", func(t *testing.T) {
		objectives, err := buildObjectives(map[string]float64{})
		require.NoError(t, err)

		quantiles, hasSum, hasCount := quantilesFromSummary(t, objectives)
		assert.Empty(t, quantiles)
		assert.True(t, hasSum)
		assert.True(t, hasCount)
	})
}
