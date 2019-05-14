package worker

import (
	"strconv"
	"time"

	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/metrics"

	workers "github.com/topfreegames/go-workers"
)

// Report sends periodic of worker reports
func Report(reporters []metrics.Reporter, period time.Duration) {
	for {
		time.Sleep(period)

		workerStats := workers.GetStats()
		for _, r := range reporters {
			reportJobsRetry(r, workerStats.Retries)
			reportQueueSizes(r, workerStats.Enqueued)
			reportJobsTotal(r, workerStats.Failed, workerStats.Processed)
		}
	}
}

func reportJobsRetry(r metrics.Reporter, retries int64) {
	err := r.ReportGauge(metrics.WorkerJobsRetry, map[string]string{}, float64(retries))
	checkReportErr(metrics.WorkerJobsRetry, err)
}

func reportQueueSizes(r metrics.Reporter, queues map[string]string) {
	for queue, size := range queues {
		tags := map[string]string{"queue": queue}
		sizeFlt, err := strconv.ParseFloat(size, 64)
		if err != nil {
			logger.Log.Errorf("queue size is not int: queue=%s size=%s", queue, size)
			continue
		}

		err = r.ReportGauge(metrics.WorkerQueueSize, tags, sizeFlt)
		checkReportErr(metrics.WorkerQueueSize, err)
	}
}

func reportJobsTotal(r metrics.Reporter, failed, processed int) {
	// "failed" and "processed" always grow up,
	// so they work as count but must be reported as gauge
	err := r.ReportGauge(metrics.WorkerJobsTotal, map[string]string{
		"status": "failed",
	}, float64(failed))
	checkReportErr(metrics.WorkerJobsTotal, err)

	err = r.ReportGauge(metrics.WorkerJobsTotal, map[string]string{
		"status": "ok",
	}, float64(processed))
	checkReportErr(metrics.WorkerJobsTotal, err)
}

func checkReportErr(metric string, err error) {
	if err != nil {
		logger.Log.Errorf("failed to report to %s: %q\n", metric, err)
	}
}
