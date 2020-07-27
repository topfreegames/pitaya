package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPercentileGenerator(t *testing.T) {
	pg := newPercentileGenerator(100)

	var result int

	result = pg.getPercentile(1)
	assert.Equal(t, result, -1)

	assert.Equal(t, pg.nSamples, 0)
	assert.Equal(t, pg.nTotalSamples, uint64(0))
	for i := 1; i <= 100; i++ {
		pg.addSample(i)
	}

	assert.Equal(t, pg.nSamples, 100)
	assert.Equal(t, pg.nTotalSamples, uint64(100))

	result = pg.getPercentile(0.2)
	assert.Equal(t, result, 20)
	result = pg.getPercentile(0)
	assert.Equal(t, result, 1)
	result = pg.getPercentile(1)
	assert.Equal(t, result, 100)
	result = pg.getPercentile(1.1)
	assert.Equal(t, result, -1)
	result = pg.getPercentile(-0.1)
	assert.Equal(t, result, -1)

	pg.addSample(1)
	assert.Equal(t, pg.nSamples, 100)
	assert.Equal(t, pg.nTotalSamples, uint64(101))
}
