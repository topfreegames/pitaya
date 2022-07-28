package metrics

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/constants"
	"testing"
)

func TestGetPrometheusReporter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := config.NewConfig()
	label := "constLabel"
	labelValue := "value"

	prometheus, err := GetPrometheusReporter("svType", cfg, map[string]string{label: labelValue})
	assert.NoError(t, err)
	assert.Equal(t, "svType", prometheus.serverType)
	assert.Equal(t, 1, len(prometheus.countReportersMap))
	assert.Equal(t, 2, len(prometheus.summaryReportersMap))
	assert.Equal(t, 10, len(prometheus.gaugeReportersMap))
}

func TestPrometheusReportSummary(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := config.NewConfig()
	value := 1.0
	responseTimeLabels := map[string]string{"route": "myRoute", "status": "myStatus", "type": "myType", "code": "200"}
	processDelayLabels := map[string]string{"route": "myRoute", "type": "myType"}

	prometheus, _ := GetPrometheusReporter("svType", cfg, map[string]string{})

	err := prometheus.ReportSummary(ResponseTime, responseTimeLabels, value)
	assert.Nil(t, err)
	err = prometheus.ReportSummary(ProcessDelay, processDelayLabels, value)
	assert.Nil(t, err)
}

func TestPrometheusReportSummaryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := config.NewConfig()
	metric := "unknownMetric"
	value := 1.0

	prometheus, err := GetPrometheusReporter("svType", cfg, map[string]string{})
	prometheus.ReportSummary(metric, map[string]string{}, value)
	assert.Error(t, constants.ErrMetricNotKnown, err)
}

func TestPrometheusReportCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := config.NewConfig()
	value := 1.0

	prometheus, _ := GetPrometheusReporter("svType", cfg, map[string]string{})
	err := prometheus.ReportCount(ExceededRateLimiting, map[string]string{}, value)
	assert.Nil(t, err)
}

func TestPrometheusReportCountError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := config.NewConfig()
	metric := "unknownMetric"
	value := 1.0

	prometheus, err := GetPrometheusReporter("svType", cfg, map[string]string{})
	prometheus.ReportCount(metric, map[string]string{}, value)
	assert.Error(t, constants.ErrMetricNotKnown, err)
}

func TestPrometheusReportGauge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := config.NewConfig()
	value := 1.0
	emptyLabels := map[string]string{}

	prometheus, _ := GetPrometheusReporter("svType", cfg, map[string]string{})

	err := prometheus.ReportGauge(ConnectedClients, emptyLabels, value)
	assert.Nil(t, err)
	err = prometheus.ReportGauge(CountServers, map[string]string{"type": "label"}, value)
	assert.Nil(t, err)
	err = prometheus.ReportGauge(ChannelCapacity, map[string]string{"channel": "label"}, value)
	assert.Nil(t, err)
	err = prometheus.ReportGauge(DroppedMessages, emptyLabels, value)
	assert.Nil(t, err)
	err = prometheus.ReportGauge(Goroutines, emptyLabels, value)
	assert.Nil(t, err)
	err = prometheus.ReportGauge(HeapSize, emptyLabels, value)
	assert.Nil(t, err)
	err = prometheus.ReportGauge(HeapObjects, emptyLabels, value)
	assert.Nil(t, err)
	err = prometheus.ReportGauge(WorkerJobsRetry, emptyLabels, value)
	assert.Nil(t, err)
	err = prometheus.ReportGauge(WorkerQueueSize, map[string]string{"queue": "label"}, value)
	assert.Nil(t, err)
	err = prometheus.ReportGauge(WorkerJobsTotal, map[string]string{"status": "label"}, value)
	assert.Nil(t, err)
}

func TestPrometheusReportGaugeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := config.NewConfig()
	metric := "unknownMetric"
	value := 1.0

	prometheus, err := GetPrometheusReporter("svType", cfg, map[string]string{})
	prometheus.ReportGauge(metric, map[string]string{}, value)
	assert.Error(t, constants.ErrMetricNotKnown, err)
}

func TestPrometheusReportEventError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := config.NewConfig()

	prometheus, err := GetPrometheusReporter("svType", cfg, map[string]string{})
	prometheus.ReportEvent("eventTitle", "eventDescription")
	assert.Error(t, constants.ErrNotImplemented, err)
}

func TestPrometheusGetMetricWithLabel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := config.NewConfig()
	value := 1.0
	responseTimeLabels := map[string]string{"route": "myRoute", "status": "myStatus", "type": "myType", "code": "200"}
	processDelayLabels := map[string]string{"route": "myRoute", "type": "myType"}

	prometheus, _ := GetPrometheusReporter("svType", cfg, map[string]string{})

	prometheus.ReportSummary(ResponseTime, responseTimeLabels, value)
	prometheus.ReportSummary(ProcessDelay, processDelayLabels, value)

	obs, err := prometheus.summaryReportersMap[ResponseTime].GetMetricWith(responseTimeLabels)
	assert.NotNil(t, obs)
	assert.Nil(t, err)

	obs, err = prometheus.summaryReportersMap[ProcessDelay].GetMetricWith(processDelayLabels)
	assert.NotNil(t, obs)
	assert.Nil(t, err)
}

func TestPrometheusGetMetricWithInvalidLabel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := config.NewConfig()
	value := 1.0
	processDelayLabels := map[string]string{"route": "myRoute", "type": "myType"}
	invalidLabels := map[string]string{"customLabel": "value"}

	prometheus, _ := GetPrometheusReporter("svType", cfg, map[string]string{})
	prometheus.ReportSummary(ProcessDelay, processDelayLabels, value)

	obs, err := prometheus.summaryReportersMap[ProcessDelay].GetMetricWith(invalidLabels)
	assert.Nil(t, obs)
	assert.NotNil(t, err)
}
