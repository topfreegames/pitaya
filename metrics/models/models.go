package models

// Summary defines a summary metric
type Summary struct {
	Subsystem  string
	Name       string
	Help       string
	Objectives map[float64]float64
	Labels     []string
}

// Gauge defines a gauge metric
type Gauge struct {
	Subsystem string
	Name      string
	Help      string
	Labels    []string
}

// Counter defines a counter metric
type Counter struct {
	Subsystem string
	Name      string
	Help      string
	Labels    []string
}

// CustomMetricsSpec has all metrics specs
type CustomMetricsSpec struct {
	Summaries []*Summary
	Gauges    []*Gauge
	Counters  []*Counter
}
