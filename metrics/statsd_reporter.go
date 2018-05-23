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
	"os"
	"strconv"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/logger"
)

// Client is the interface to required dogstatsd functions
type Client interface {
	Gauge(name string, value float64, tags []string, rate float64) error
	Timing(name string, value time.Duration, tags []string, rate float64) error
}

// StatsdReporter sends application metrics to statsd
type StatsdReporter struct {
	client     Client
	rate       float64
	serverType string
	hostname   string
}

// NewStatsdReporter returns an instance of statsd reportar and an error if something fails
func NewStatsdReporter(config *config.Config, serverType string, clientOrNil ...Client) (*StatsdReporter, error) {
	host := config.GetString("pitaya.metrics.statsd.host")
	prefix := config.GetString("pitaya.metrics.statsd.prefix")
	rate, err := strconv.ParseFloat(config.GetString("pitaya.metrics.statsd.rate"), 64)
	if err != nil {
		return nil, err
	}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	sr := &StatsdReporter{
		rate:       rate,
		serverType: serverType,
		hostname:   hostname,
	}

	if len(clientOrNil) > 0 {
		sr.client = clientOrNil[0]
	} else {
		c, err := statsd.New(host)
		if err != nil {
			return nil, err
		}
		c.Namespace = prefix
		sr.client = c
	}
	return sr, nil
}

// ReportLatency sends latency reports to statsd
func (s *StatsdReporter) ReportLatency(value time.Duration, route, typ string, errored bool) error {
	tags := []string{
		fmt.Sprintf("route:%s", route),
		fmt.Sprintf("type:%s", typ),
		fmt.Sprintf("error:%t", errored),
		fmt.Sprintf("serverType:%s", s.serverType),
		fmt.Sprintf("hostname:%s", s.hostname),
	}

	err := s.client.Timing("response_time_ms", value, tags, s.rate)
	if err != nil {
		logger.Log.Errorf("failed to report latency: %q", err)
	}

	return err
}

// ReportCount sends count reports to statsd
func (s *StatsdReporter) ReportCount(value int, metric string, tags ...string) error {
	fullTags := []string{
		fmt.Sprintf("serverType:%s", s.serverType),
		fmt.Sprintf("hostname:%s", s.hostname),
	}
	if len(tags) > 0 {
		fullTags = append(fullTags, tags...)
	}

	err := s.client.Gauge(metric, float64(value), fullTags, s.rate)
	if err != nil {
		logger.Log.Errorf("failed to report count: %q", err)
	}

	return err
}
