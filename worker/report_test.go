package worker

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/topfreegames/pitaya/metrics"
	"github.com/topfreegames/pitaya/metrics/mocks"
)

func TestReportJobsRetry(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockReporter := mocks.NewMockReporter(ctrl)
		mockReporter.EXPECT().ReportGauge(
			metrics.WorkerJobsRetry,
			map[string]string{},
			float64(10))

		reportJobsRetry(mockReporter, 10)
	})

	t.Run("error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockReporter := mocks.NewMockReporter(ctrl)
		mockReporter.EXPECT().ReportGauge(
			metrics.WorkerJobsRetry,
			map[string]string{},
			float64(10)).Return(errors.New("err"))

		reportJobsRetry(mockReporter, 10)
	})
}

func TestReportQueueSizes(t *testing.T) {
	t.Parallel()

	var (
		queue1 = "queuename1"
		queue2 = "queuename2"
	)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReporter := mocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().ReportGauge(
		metrics.WorkerQueueSize,
		map[string]string{"queue": queue1},
		float64(10))
	mockReporter.EXPECT().ReportGauge(
		metrics.WorkerQueueSize,
		map[string]string{"queue": queue2},
		float64(20))

	reportQueueSizes(mockReporter, map[string]string{
		queue1: "10",
		queue2: "20",
	})
}

func TestReportJobsTotal(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReporter := mocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().ReportGauge(
		metrics.WorkerJobsTotal,
		map[string]string{"status": "failed"},
		float64(10))

	mockReporter.EXPECT().ReportGauge(
		metrics.WorkerJobsTotal,
		map[string]string{"status": "ok"},
		float64(20))

	reportJobsTotal(mockReporter, 10, 20)
}
