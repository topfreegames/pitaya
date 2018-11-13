package services

import (
	"context"

	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/examples/demo/custom_metrics/messages"
)

// Room server
type Room struct {
	component.Base
}

// SetCounter sets custom my_counter
func (*Room) SetCounter(
	ctx context.Context,
	arg *messages.SetCounterArg,
) (*messages.Response, error) {
	counterMetricName := "my_counter"

	for _, reporter := range pitaya.GetMetricsReporters() {
		reporter.ReportCount(counterMetricName, map[string]string{
			"tag1": arg.Tag1,
			"tag2": arg.Tag2,
		}, arg.Value)
	}

	return messages.OKResponse(), nil
}

// SetGauge1 sets custom my_gauge_1
func (*Room) SetGauge1(
	ctx context.Context,
	arg *messages.SetGaugeArg,
) (*messages.Response, error) {
	counterMetricName := "my_gauge_1"

	for _, reporter := range pitaya.GetMetricsReporters() {
		reporter.ReportGauge(counterMetricName, map[string]string{
			"tag1": arg.Tag,
		}, arg.Value)
	}

	return messages.OKResponse(), nil
}

// SetGauge2 sets custom my_gauge_2
func (*Room) SetGauge2(
	ctx context.Context,
	arg *messages.SetGaugeArg,
) (*messages.Response, error) {
	counterMetricName := "my_gauge_2"

	for _, reporter := range pitaya.GetMetricsReporters() {
		reporter.ReportGauge(counterMetricName, map[string]string{
			"tag2": arg.Tag,
		}, arg.Value)
	}

	return messages.OKResponse(), nil
}

// SetSummary sets custom my_summary
func (*Room) SetSummary(
	ctx context.Context,
	arg *messages.SetSummaryArg,
) (*messages.Response, error) {
	counterMetricName := "my_summary"

	for _, reporter := range pitaya.GetMetricsReporters() {
		reporter.ReportSummary(counterMetricName, map[string]string{
			"tag1": arg.Tag,
		}, arg.Value)
	}

	return messages.OKResponse(), nil
}
