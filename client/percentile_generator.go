package client

import (
	"math/rand"
	"sort"
	"time"
)

const (
	// MAX_PERCENTILE_GENERATOR_SAMPLES is the max samples that a PercentileGeneretor will store
	MAX_PERCENTILE_GENERATOR_SAMPLES = 1000
)

// percentileGenerator is used to collect samples and then get a percentile breakdown of the data.
// This class can be used even if the number of data points you collect grows
// beyond the number you want to store in memory, by keeping a random
// subsample.  Note that if the table is filled and we have to resort
// to sub-sampling, that the resulting sample will be based on the entire
// data set you provide.  It will not be biased towards the first or last samples.
// Source: https://github.com/ValveSoftware/GameNetworkingSockets/blob/f3f9c853d45d9f5aaa98f7eb53d895d514c57f19/src/common/percentile_generator.h
type percentileGenerator struct {
	maxSamples    int
	samples       []int
	isSorted      bool
	nSamples      int
	nTotalSamples uint64
}

// newPercentileGenerator creates a new PercentileGenerator
func newPercentileGenerator(maxSamples int) *percentileGenerator {
	source := rand.NewSource(time.Now().UnixNano())
	rand.Seed(source.Int63())
	return &percentileGenerator{
		maxSamples:    maxSamples,
		samples:       make([]int, maxSamples),
		isSorted:      false,
		nSamples:      0,
		nTotalSamples: 0,
	}
}

// AddSample adds a new sample to PercentileGenerator
func (pg *percentileGenerator) addSample(sample int) {

	if pg.nSamples < pg.maxSamples {
		pg.samples[pg.nSamples] = sample
		pg.nSamples++
		pg.isSorted = false
	} else {
		index := rand.Intn(pg.maxSamples)
		pg.samples[index] = sample
		pg.isSorted = false
	}
	pg.nTotalSamples++
}

// GetPercentile returns a estimate of the nth percentile. The percentile should be in the range (0, 1)
func (pg *percentileGenerator) getPercentile(percentile float32) int {
	if percentile < 0 || percentile > 1 || pg.nSamples == 0 {
		return -1
	}

	if !pg.isSorted {
		sort.Ints(pg.samples)
		pg.isSorted = true
	}

	fltIdx := percentile*float32(pg.nSamples-1) + float32(pg.maxSamples-pg.nSamples)
	idx := int(fltIdx)

	if idx <= 0 {
		return pg.samples[0]
	}
	if idx >= pg.maxSamples-1 {
		return pg.samples[pg.maxSamples-1]
	}

	if fltIdx != float32(idx) && idx+1 < pg.maxSamples {
		return int((pg.samples[idx] + pg.samples[idx+1]) / 2)
	}

	return pg.samples[idx]

}
