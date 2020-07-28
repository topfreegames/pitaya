package client

// Metrics stores client's network metrics, such as
// pingTracker, to handle ping information; jitter track to handle ping variations and A
type metrics struct {
	ping             *pingTracker
	jitter           *jitterTracker
	nPacketsSent     int64
	nPacketsReceived int64
	nPacketsLost     int64
}

// NewMetrics return a new Metrics struct
func newMetrics() *metrics {
	return &metrics{
		ping:             newPingTracker(),
		jitter:           newJitterTracker(),
		nPacketsSent:     0,
		nPacketsReceived: 0,
		nPacketsLost:     0,
	}
}

func (m *metrics) addMeasurement(pingMS int) {
	m.nPacketsReceived++
	m.ping.receivedPing(pingMS)
	jitter := abs(m.ping.lastPings[0] - m.ping.lastPings[1])
	m.jitter.addSample(jitter)
}

func (m *metrics) GetCurrentPing() int {
	return m.ping.smoothedPing
}

func (m *metrics) GetCurrentJitter() int {
	if m.jitter.jitterHistogram.nSamples == 0 {
		return -1
	}
	return m.jitter.current
}

func (m *metrics) GetPingPercentile(percentile float32) int {
	return m.ping.pingHistogram.getPercentile(percentile)
}

func (m *metrics) GetJitterPercentile(percentile float32) int {
	return m.jitter.jitterHistogram.getPercentile(percentile)
}

func (m *metrics) GetPackagesSent() int64 {
	return m.nPacketsSent
}

func (m *metrics) GetPackagesConfirmed() int64 {
	return m.nPacketsReceived
}

func (m *metrics) GetPackagesLost() int64 {
	return m.nPacketsLost
}

// pingTracker struct used to track ping values
type pingTracker struct {
	validPings    int
	smoothedPing  int // Take the average of the two best last ping measurements, ignoring a occasional ping spike
	lastPings     []int
	pingHistogram *percentileGenerator
}

// newPingTracker returns a new PingTracker struct
func newPingTracker() *pingTracker {
	return &pingTracker{
		validPings:    0,
		smoothedPing:  -1,
		lastPings:     make([]int, 3),
		pingHistogram: newPercentileGenerator(MAX_PERCENTILE_GENERATOR_SAMPLES),
	}
}

// ReceivedPing is called when a new ping measurement is received
func (pt *pingTracker) receivedPing(ping int) {
	if ping < 0 {
		return
	}

	pt.pingHistogram.addSample(ping)
	pt.lastPings[2] = pt.lastPings[1]
	pt.lastPings[1] = pt.lastPings[0]
	pt.lastPings[0] = ping

	maxPing := pt.max()
	switch pt.validPings {
	case 2:
		pt.smoothedPing = (pt.lastPings[0] + pt.lastPings[1] + pt.lastPings[2] - maxPing) >> 1
	case 1:
		pt.smoothedPing = (pt.lastPings[0] + pt.lastPings[1]) >> 1
		pt.validPings++
	case 0:
		pt.smoothedPing = pt.lastPings[0]
		pt.validPings++
	}
}

func (pt *pingTracker) max() int {
	maxPing := -1
	for i := 0; i < len(pt.lastPings); i++ {
		if pt.lastPings[i] > maxPing {
			maxPing = pt.lastPings[i]
		}
	}
	return maxPing
}

type jitterTracker struct {
	current         int
	highest         int
	jitterHistogram *percentileGenerator
}

func newJitterTracker() *jitterTracker {
	return &jitterTracker{
		current:         -1,
		highest:         -1,
		jitterHistogram: newPercentileGenerator(MAX_PERCENTILE_GENERATOR_SAMPLES),
	}
}

func (jt *jitterTracker) addSample(jitterMs int) {
	if jitterMs > jt.highest {
		jt.highest = jitterMs
	}
	jt.current = jitterMs
	jt.jitterHistogram.addSample(jitterMs)
}

func abs(num int) int {
	if num < 0 {
		return -num
	}
	return num
}
