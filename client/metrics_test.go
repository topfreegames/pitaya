package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {

	metrics := newMetrics()

	// No mesurements
	assert.Equal(t, -1, metrics.GetCurrentPing())
	assert.Equal(t, -1, metrics.GetCurrentJitter())

	metrics.addMeasurement(30)
	assert.Equal(t, 30, metrics.GetCurrentPing()) // Only ping collected
	assert.Equal(t, 30, metrics.GetCurrentJitter())
	assert.Equal(t, int64(1), metrics.GetPackagesConfirmed())

	metrics.addMeasurement(10)
	assert.Equal(t, 20, metrics.GetCurrentPing()) // Smoothed Ping
	assert.Equal(t, 20, metrics.GetCurrentJitter())
	assert.Equal(t, int64(2), metrics.GetPackagesConfirmed())

	metrics.addMeasurement(50)
	assert.Equal(t, 20, metrics.GetCurrentPing()) // Smoothed Ping
	assert.Equal(t, 40, metrics.GetCurrentJitter())
	assert.Equal(t, int64(3), metrics.GetPackagesConfirmed())

	metrics.addMeasurement(100)
	metrics.addMeasurement(60)
	assert.Equal(t, 55, metrics.GetCurrentPing())
	assert.Equal(t, 40, metrics.GetCurrentJitter())
	assert.Equal(t, int64(5), metrics.GetPackagesConfirmed())

	// Pings so far (sorted): [10, 30, 50, 60, 100]
	// Jitters so far (sorted): [20, 30, 40, 40, 50]
	assert.Equal(t, 10, metrics.GetPingPercentile(0))
	assert.Equal(t, 20, metrics.GetJitterPercentile(0))
	assert.Equal(t, 50, metrics.GetPingPercentile(0.5))
	assert.Equal(t, 40, metrics.GetJitterPercentile(0.5))
	assert.Equal(t, 100, metrics.GetPingPercentile(1))
	assert.Equal(t, 50, metrics.GetJitterPercentile(1))

}
