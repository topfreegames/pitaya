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
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/constants"
)

var (
	prometheusReporter *PrometheusReporter
	once               sync.Once
)

// PrometheusReporter reports metrics to prometheus
type PrometheusReporter struct {
	serverType          string
	game                string
	countReportersMap   map[string]*prometheus.CounterVec
	summaryReportersMap map[string]*prometheus.SummaryVec
	gaugeReportersMap   map[string]*prometheus.GaugeVec
	additionalLabels    map[string]string
}

func (p *PrometheusReporter) registerCustomMetrics(
	constLabels map[string]string,
	additionalLabelsKeys []string,
	spec *CustomMetricsSpec,
) {
	for _, summary := range spec.Summaries {
		p.summaryReportersMap[summary.Name] = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:   "pitaya",
				Subsystem:   summary.Subsystem,
				Name:        summary.Name,
				Help:        summary.Help,
				Objectives:  summary.Objectives,
				ConstLabels: constLabels,
			},
			append(additionalLabelsKeys, summary.Labels...),
		)
	}

	for _, gauge := range spec.Gauges {
		p.gaugeReportersMap[gauge.Name] = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "pitaya",
				Subsystem:   gauge.Subsystem,
				Name:        gauge.Name,
				Help:        gauge.Help,
				ConstLabels: constLabels,
			},
			append(additionalLabelsKeys, gauge.Labels...),
		)
	}

	for _, counter := range spec.Counters {
		p.countReportersMap[counter.Name] = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "pitaya",
				Subsystem:   counter.Subsystem,
				Name:        counter.Name,
				Help:        counter.Help,
				ConstLabels: constLabels,
			},
			append(additionalLabelsKeys, counter.Labels...),
		)
	}
}

func (p *PrometheusReporter) registerMetrics(
	constLabels, additionalLabels map[string]string,
	spec *CustomMetricsSpec,
) {

	constLabels["game"] = p.game
	constLabels["serverType"] = p.serverType

	p.additionalLabels = additionalLabels
	additionalLabelsKeys := make([]string, 0, len(additionalLabels))
	for key := range additionalLabels {
		additionalLabelsKeys = append(additionalLabelsKeys, key)
	}

	p.registerCustomMetrics(constLabels, additionalLabelsKeys, spec)

	// HandlerResponseTimeMs summary
	p.summaryReportersMap[ResponseTime] = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:   "pitaya",
			Subsystem:   "handler",
			Name:        ResponseTime,
			Help:        "the time to process a msg in nanoseconds",
			Objectives:  map[float64]float64{0.7: 0.02, 0.95: 0.005, 0.99: 0.001},
			ConstLabels: constLabels,
		},
		append([]string{"route", "status", "type", "code"}, additionalLabelsKeys...),
	)

	// ProcessDelay summary
	p.summaryReportersMap[ProcessDelay] = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:   "pitaya",
			Subsystem:   "handler",
			Name:        ProcessDelay,
			Help:        "the delay to start processing a msg in nanoseconds",
			Objectives:  map[float64]float64{0.7: 0.02, 0.95: 0.005, 0.99: 0.001},
			ConstLabels: constLabels,
		},
		append([]string{"route", "type"}, additionalLabelsKeys...),
	)

	// ConnectedClients gauge
	p.gaugeReportersMap[ConnectedClients] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "pitaya",
			Subsystem:   "acceptor",
			Name:        ConnectedClients,
			Help:        "the number of clients connected right now",
			ConstLabels: constLabels,
		},
		additionalLabelsKeys,
	)

	p.gaugeReportersMap[CountServers] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "pitaya",
			Subsystem:   "service_discovery",
			Name:        CountServers,
			Help:        "the number of discovered servers by service discovery",
			ConstLabels: constLabels,
		},
		append([]string{"type"}, additionalLabelsKeys...),
	)

	p.gaugeReportersMap[ChannelCapacity] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "pitaya",
			Subsystem:   "channel",
			Name:        ChannelCapacity,
			Help:        "the available capacity of the channel",
			ConstLabels: constLabels,
		},
		append([]string{"channel"}, additionalLabelsKeys...),
	)

	p.gaugeReportersMap[DroppedMessages] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "pitaya",
			Subsystem:   "rpc_server",
			Name:        DroppedMessages,
			Help:        "the number of rpc server dropped messages (messages that are not handled)",
			ConstLabels: constLabels,
		},
		additionalLabelsKeys,
	)

	p.gaugeReportersMap[Goroutines] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "pitaya",
			Subsystem:   "sys",
			Name:        Goroutines,
			Help:        "the current number of goroutines",
			ConstLabels: constLabels,
		},
		additionalLabelsKeys,
	)

	p.gaugeReportersMap[HeapSize] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "pitaya",
			Subsystem:   "sys",
			Name:        HeapSize,
			Help:        "the current heap size",
			ConstLabels: constLabels,
		},
		additionalLabelsKeys,
	)

	p.gaugeReportersMap[HeapObjects] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "pitaya",
			Subsystem:   "sys",
			Name:        HeapObjects,
			Help:        "the current number of allocated heap objects",
			ConstLabels: constLabels,
		},
		additionalLabelsKeys,
	)

	p.gaugeReportersMap[WorkerJobsRetry] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "pitaya",
			Subsystem:   "worker",
			Name:        WorkerJobsRetry,
			Help:        "the current number of job retries",
			ConstLabels: constLabels,
		},
		additionalLabelsKeys,
	)

	p.gaugeReportersMap[WorkerQueueSize] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "pitaya",
			Subsystem:   "worker",
			Name:        WorkerQueueSize,
			Help:        "the current queue size",
			ConstLabels: constLabels,
		},
		append([]string{"queue"}, additionalLabelsKeys...),
	)

	p.gaugeReportersMap[WorkerJobsTotal] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "pitaya",
			Subsystem:   "worker",
			Name:        WorkerJobsTotal,
			Help:        "the total executed jobs",
			ConstLabels: constLabels,
		},
		append([]string{"status"}, additionalLabelsKeys...),
	)

	p.countReportersMap[ExceededRateLimiting] = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "pitaya",
			Subsystem:   "acceptor",
			Name:        ExceededRateLimiting,
			Help:        "the number of blocked requests by exceeded rate limiting",
			ConstLabels: constLabels,
		},
		additionalLabelsKeys,
	)

	toRegister := make([]prometheus.Collector, 0)
	for _, c := range p.countReportersMap {
		toRegister = append(toRegister, c)
	}

	for _, c := range p.gaugeReportersMap {
		toRegister = append(toRegister, c)
	}

	for _, c := range p.summaryReportersMap {
		toRegister = append(toRegister, c)
	}

	prometheus.MustRegister(toRegister...)
}

// GetPrometheusReporter gets the prometheus reporter singleton
func GetPrometheusReporter(
	serverType string,
	config *config.Config,
	constLabels map[string]string,
) (*PrometheusReporter, error) {
	var (
		port             = config.GetInt("pitaya.metrics.prometheus.port")
		game             = config.GetString("pitaya.game")
		additionalLabels = config.GetStringMapString("pitaya.metrics.additionalTags")
	)

	spec, err := NewCustomMetricsSpec(config)
	if err != nil {
		return nil, err
	}

	once.Do(func() {
		prometheusReporter = &PrometheusReporter{
			serverType:          serverType,
			game:                game,
			countReportersMap:   make(map[string]*prometheus.CounterVec),
			summaryReportersMap: make(map[string]*prometheus.SummaryVec),
			gaugeReportersMap:   make(map[string]*prometheus.GaugeVec),
		}
		prometheusReporter.registerMetrics(constLabels, additionalLabels, spec)
		http.Handle("/metrics", promhttp.Handler())
		go (func() {
			log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
		})()
	})

	return prometheusReporter, nil
}

// ReportSummary reports a summary metric
func (p *PrometheusReporter) ReportSummary(metric string, labels map[string]string, value float64) error {
	sum := p.summaryReportersMap[metric]
	if sum != nil {
		labels = p.ensureLabels(labels)
		sum.With(labels).Observe(value)
		return nil
	}
	return constants.ErrMetricNotKnown
}

// ReportCount reports a summary metric
func (p *PrometheusReporter) ReportCount(metric string, labels map[string]string, count float64) error {
	cnt := p.countReportersMap[metric]
	if cnt != nil {
		labels = p.ensureLabels(labels)
		cnt.With(labels).Add(count)
		return nil
	}
	return constants.ErrMetricNotKnown
}

// ReportGauge reports a gauge metric
func (p *PrometheusReporter) ReportGauge(metric string, labels map[string]string, value float64) error {
	g := p.gaugeReportersMap[metric]
	if g != nil {
		labels = p.ensureLabels(labels)
		g.With(labels).Set(value)
		return nil
	}
	return constants.ErrMetricNotKnown
}

// ensureLabels checks if labels contains the additionalLabels values,
// otherwise adds them with the default values
func (p *PrometheusReporter) ensureLabels(labels map[string]string) map[string]string {
	for key, defaultVal := range p.additionalLabels {
		if _, ok := labels[key]; !ok {
			labels[key] = defaultVal
		}
	}

	return labels
}
