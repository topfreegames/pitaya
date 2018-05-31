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
	"github.com/topfreegames/pitaya/constants"
)

var (
	// ResponseTime reports the response time of handlers and rpc
	ResponseTime = "response_time_ns"
	// ConnectedClients represents the number of current connected clients in frontend servers
	ConnectedClients = "connected_clients"
	// CountServers counts the number of servers of different types
	CountServers = "count_servers"
	// ChannelCapacity represents the capacity of a channel (available slots)
	ChannelCapacity = "channel_capacity"
	// DroppedMessages reports the number of dropped messages in rpc server (messages that will not be handled)
	DroppedMessages    = "dropped_messages"
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
}

func (p *PrometheusReporter) registerMetrics() {
	// HanadlerResponseTimeMs summaary
	p.summaryReportersMap[ResponseTime] = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "pitaya",
			Subsystem:  "handler",
			Name:       ResponseTime,
			Help:       "the time to process a msg in nanoseconds",
			Objectives: map[float64]float64{0.7: 0.02, 0.95: 0.005, 0.99: 0.001},
			ConstLabels: map[string]string{
				"game":       p.game,
				"serverType": p.serverType,
			},
		},
		[]string{"route", "status", "type"},
	)

	// ConnectedClients gauge
	p.gaugeReportersMap[ConnectedClients] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pitaya",
			Subsystem: "acceptor",
			Name:      ConnectedClients,
			Help:      "the number of clients connected right now",
			ConstLabels: map[string]string{
				"game":       p.game,
				"serverType": p.serverType,
			},
		},
		[]string{},
	)

	p.gaugeReportersMap[CountServers] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pitaya",
			Subsystem: "service_discovery",
			Name:      CountServers,
			Help:      "the number of discovered servers by service discovery",
			ConstLabels: map[string]string{
				"game":       p.game,
				"serverType": p.serverType,
			},
		},
		[]string{"type"},
	)

	p.gaugeReportersMap[ChannelCapacity] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pitaya",
			Subsystem: "channel",
			Name:      ChannelCapacity,
			Help:      "the available capacity of the channel",
			ConstLabels: map[string]string{
				"game":       p.game,
				"serverType": p.serverType,
			},
		},
		[]string{"channel"},
	)

	p.gaugeReportersMap[DroppedMessages] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pitaya",
			Subsystem: "rpc_server",
			Name:      DroppedMessages,
			Help:      "the number of rpc server dropped messages (messages that are not handled)",
			ConstLabels: map[string]string{
				"game":       p.game,
				"serverType": p.serverType,
			},
		},
		[]string{},
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
func GetPrometheusReporter(serverType string, game string, port int) *PrometheusReporter {
	once.Do(func() {
		prometheusReporter = &PrometheusReporter{
			serverType:          serverType,
			game:                game,
			countReportersMap:   make(map[string]*prometheus.CounterVec),
			summaryReportersMap: make(map[string]*prometheus.SummaryVec),
			gaugeReportersMap:   make(map[string]*prometheus.GaugeVec),
		}
		prometheusReporter.registerMetrics()
		http.Handle("/metrics", prometheus.Handler())
		go (func() {
			log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
		})()
	})
	return prometheusReporter
}

// ReportSummary reports a summary metric
func (p *PrometheusReporter) ReportSummary(metric string, tags map[string]string, value float64) error {
	sum := p.summaryReportersMap[metric]
	if sum != nil {
		sum.With(tags).Observe(value)
		return nil
	}
	return constants.ErrMetricNotKnown
}

// ReportCount reports a summary metric
func (p *PrometheusReporter) ReportCount(metric string, tags map[string]string, count float64) error {
	cnt := p.countReportersMap[metric]
	if cnt != nil {
		cnt.With(tags).Add(count)
		return nil
	}
	return constants.ErrMetricNotKnown
}

// ReportGauge reports a gauge metric
func (p *PrometheusReporter) ReportGauge(metric string, tags map[string]string, value float64) error {
	g := p.gaugeReportersMap[metric]
	if g != nil {
		g.With(tags).Set(value)
		return nil
	}
	return constants.ErrMetricNotKnown
}
