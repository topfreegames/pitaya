package pitaya

import (
	"errors"

	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/metrics"
)

type pitayaMetrics struct {
	RequestResponseTime *metrics.Metric
	TimeoutRequests     *metrics.Metric
	TagsAndMeta         *metrics.TagsAndMeta
}

// registerMetrics registers the metrics for the mqtt module in the metrics registry
func registerMetrics(vu modules.VU) (pitayaMetrics, error) {
	p := pitayaMetrics{}
	env := vu.InitEnv()
	if env == nil {
		return p, errors.New("missing env")
	}
	registry := env.Registry
	if registry == nil {
		return p, errors.New("missing registry")
	}

	var err error
	p.RequestResponseTime, err = registry.NewMetric("pitaya_client_request_duration_ms", metrics.Trend)
	if err != nil {
		return p, err
	}

	p.TimeoutRequests, err = registry.NewMetric("pitaya_client_request_timeout_count", metrics.Counter)
	if err != nil {
		return p, err
	}

	p.TagsAndMeta = &metrics.TagsAndMeta{
		Tags: registry.RootTagSet(),
	}
	return p, nil
}
