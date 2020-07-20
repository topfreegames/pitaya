package client

// PingTracker struct used to track ping values
type PingTracker struct {
	validPings   int
	SmoothedPing int64 // Take the average of the two best last ping measurements, ignoring a occasional ping spike
	lastPings    []int64
	HighestPing  int64
	LowestPing   int64
}

// NewPingTracker returns a new PingTracker struct
func NewPingTracker() *PingTracker {
	return &PingTracker{
		validPings:   0,
		SmoothedPing: -1,
		lastPings:    make([]int64, 3),
		HighestPing:  -1,
		LowestPing:   -1,
	}
}

// ReceivedPing is called when a new ping measurement is received
func (pt *PingTracker) ReceivedPing(ping int64) {
	if ping < 0 {
		return
	}

	pt.lastPings[2] = pt.lastPings[1]
	pt.lastPings[1] = pt.lastPings[0]
	pt.lastPings[0] = ping

	maxPing := pt.max()
	switch pt.validPings {
	case 3:
		pt.SmoothedPing = (pt.lastPings[0] + pt.lastPings[2] + pt.lastPings[2] - maxPing) >> 1
	case 2:
		pt.SmoothedPing = (pt.lastPings[0] + pt.lastPings[1]) >> 1
		pt.validPings++
	case 1:
		pt.SmoothedPing = pt.lastPings[0]
		pt.validPings++
	}
}

func (pt *PingTracker) max() int64 {
	maxPing := int64(-1)
	for i := 0; i < len(pt.lastPings); i++ {
		if pt.lastPings[i] > maxPing {
			maxPing = pt.lastPings[i]
		}
	}
	return maxPing
}

// Metrics stores client's network metrics
type Metrics struct {
	Ping *PingTracker
}

// NewMetrics return a new Metrics struct
func NewMetrics() *Metrics {
	return &Metrics{
		Ping: NewPingTracker(),
	}
}
