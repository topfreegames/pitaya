using System.Collections.Generic;
using System.Linq;
using Prometheus;

namespace NPitaya.Metrics
{
    public class CustomMetricsSpec
    {
        public List<CustomSummary> Summaries = new List<CustomSummary>();
        public List<CustomGauge> Gauges = new List<CustomGauge>();
        public List<CustomCounter> Counters = new List<CustomCounter>();
    }

    public class CustomSummary
    {
        public string Subsystem;
        public string Name;
        public string Help;
        public Dictionary<double, double> Objectives = new Dictionary<double, double>();
        public List<string> Labels = new List<string>();
    }

    public class CustomGauge
    {
        public string Subsystem;
        public string Name;
        public string Help;
        public List<string> Labels = new List<string>();
    }

    public class CustomCounter
    {
        public string Subsystem;
        public string Name;
        public string Help;
        public List<string> Labels = new List<string>();
    }
    public class PrometheusMetricsReporter: IMetricsReporter
    {
        private Dictionary<string, string> _additionalLabels;
        private Dictionary<string, string> _constantLabels;
        private Dictionary<string, Counter> _countReportersMap;
        private Dictionary<string, Summary> _summaryReportersMap;
        private Dictionary<string, Gauge> _gaugeReportersMap;

        public PrometheusMetricsReporter(string serverType, string game, int port, Dictionary<string, string> constantLabels = null, Dictionary<string, string> additionalLabels = null, CustomMetricsSpec customMetricsSpec = null)
        {
            _constantLabels = constantLabels ?? new Dictionary<string, string>();
            _constantLabels["game"] = game;
            _constantLabels["serverType"] = serverType;
            _additionalLabels = additionalLabels ?? new Dictionary<string, string>();
            _countReportersMap = new Dictionary<string, Counter>();
            _summaryReportersMap = new Dictionary<string, Summary>();
            _gaugeReportersMap = new Dictionary<string, Gauge>();

            if (customMetricsSpec != null)
            {
                _registerCustomMetrics(customMetricsSpec);
            }

            _registerMetrics();

            var prometheusServer = new MetricServer(port: port);
            prometheusServer.Start();
        }

        private void _registerCustomMetrics(CustomMetricsSpec customMetricsSpec)
        {
            foreach (var summary in customMetricsSpec.Summaries)
            {
                _addSummaryReporter(
                    Constants.PitayaKey,
                    summary.Subsystem,
                    summary.Name,
                    summary.Help,
                    summary.Labels.ToArray(),
                    summary.Objectives);
            }

            foreach (var gauge in customMetricsSpec.Gauges)
            {
                _addGaugeReporter(
                    Constants.PitayaKey,
                    gauge.Subsystem,
                    gauge.Name,
                    gauge.Help,
                    gauge.Labels.ToArray());
            }

            foreach (var counter in customMetricsSpec.Counters)
            {
                _addCounterReporter(
                    Constants.PitayaKey,
                    counter.Subsystem,
                    counter.Name,
                    counter.Help,
                    counter.Labels.ToArray());
            }
        }

        private void _registerMetrics()
        {
            _addSummaryReporter(
                Constants.PitayaKey,
                "handler",
                Constants.ResponseTimeMetricKey,
                "the time to process a msg in nanoseconds",
                new[]{"route", "status", "type", "code"});

            _addSummaryReporter(
                Constants.PitayaKey,
                "handler",
                Constants.ProccessDelayMetricKey,
                "the delay to start processing a msg in nanoseconds",
                new[]{"route", "type"});

            _addGaugeReporter(
                Constants.PitayaKey,
                "acceptor",
                Constants.ConnectedClientsMetricKey,
                "the number of clients connected right now",
                new string[]{});

            _addGaugeReporter(
                Constants.PitayaKey,
                "service_discovery",
                Constants.CountServersMetricKey,
                "the number of discovered servers by service discovery",
                new[]{"type"});

            _addGaugeReporter(
                Constants.PitayaKey,
                "channel",
                Constants.ChannelCapacityMetricKey,
                "the available capacity of the channel",
                new[]{"channel"});

            _addGaugeReporter(
                Constants.PitayaKey,
                "rpc_server",
                Constants.DroppedMessagesMetricKey,
                "the number of rpc server dropped messages (messages that are not handled)",
                new string[]{});

            _addGaugeReporter(
                Constants.PitayaKey,
                "sys",
                Constants.GoroutinesMetricKey,
                "the current number of goroutines",
                new string[]{});

            _addGaugeReporter(
                Constants.PitayaKey,
                "sys",
                Constants.HeapSizeMetricKey,
                "the current heap size",
                new string[]{});

            _addGaugeReporter(
                Constants.PitayaKey,
                "sys",
                Constants.HeapObjectsMetricKey,
                "the current number of allocated heap objects",
                new string[]{});

            _addGaugeReporter(
                Constants.PitayaKey,
                "worker",
                Constants.WorkerJobsRetryMetricKey,
                "the current number of job retries",
                new string[]{});

            _addGaugeReporter(
                Constants.PitayaKey,
                "worker",
                Constants.WorkerQueueSizeMetricKey,
                "the current queue size",
                new[]{"queue"});

            _addGaugeReporter(
                Constants.PitayaKey,
                "worker",
                Constants.WorkerJobsTotalMetricKey,
                "the total executed jobs",
                new[]{"status"});

            _addCounterReporter(
                Constants.PitayaKey,
                "acceptor",
                Constants.ExceededRateLimitingMetricKey,
                "the number of blocked requests by exceeded rate limiting",
                new string[]{});
        }

        private void _addSummaryReporter(string metricNamespace, string metricSubsystem, string metricName, string metricHelp, string[] metricLabels, Dictionary<double, double> objectives = null)
        {
            var allLabels = new List<string>();
            allLabels.AddRange(_constantLabels.Keys.ToArray());
            allLabels.AddRange(_additionalLabels.Keys.ToArray());
            allLabels.AddRange(metricLabels);
            var summaryObjectives = new List<QuantileEpsilonPair>();
            if (objectives != null)
            {
                foreach (var kv in objectives)
                {
                    summaryObjectives.Add(new QuantileEpsilonPair(kv.Key, kv.Value));
                }
            }
            else
            {
                summaryObjectives.Add(new QuantileEpsilonPair(0.7, 0.02));
                summaryObjectives.Add(new QuantileEpsilonPair(0.95, 0.005));
                summaryObjectives.Add(new QuantileEpsilonPair(0.99, 0.001));
            }


            var summary = Prometheus.Metrics.CreateSummary(metricNamespace + "_" + metricSubsystem + "_" + metricName, metricHelp, new SummaryConfiguration
            {
                Objectives = summaryObjectives,
                LabelNames = allLabels.ToArray(),
            });
            _summaryReportersMap[metricName] = summary;
        }

        private void _addCounterReporter(string metricNamespace, string metricSubsystem, string metricName, string metricHelp, string[] metricLabels)
        {
            var allLabels = new List<string>();
            allLabels.AddRange(_constantLabels.Keys.ToArray());
            allLabels.AddRange(_additionalLabels.Keys.ToArray());
            allLabels.AddRange(metricLabels);
            var counter = Prometheus.Metrics.CreateCounter(metricNamespace + "_" + metricSubsystem + "_" + metricName, metricHelp, new CounterConfiguration
            {
                LabelNames = allLabels.ToArray(),
            });
            _countReportersMap[metricName] = counter;
        }

        private void _addGaugeReporter(string metricNamespace, string metricSubsystem, string metricName, string metricHelp, string[] metricLabels)
        {
            var allLabels = new List<string>();
            allLabels.AddRange(_constantLabels.Keys.ToArray());
            allLabels.AddRange(_additionalLabels.Keys.ToArray());
            allLabels.AddRange(metricLabels);
            var gauge = Prometheus.Metrics.CreateGauge(metricNamespace + "_" + metricSubsystem + "_" + metricName, metricHelp, new GaugeConfiguration
            {
                LabelNames = allLabels.ToArray(),
            });
            _gaugeReportersMap[metricName] = gauge;
        }

        public void ReportCount(string metricKey, Dictionary<string, string> tags, double value)
        {
            if (!_countReportersMap.TryGetValue(metricKey, out var counter)) return;
            var labelValues = _ensureLabels(tags, counter.LabelNames);
            counter.WithLabels(labelValues).Inc(value);
        }

        public void ReportGauge(string metricKey, Dictionary<string, string> tags, double value)
        {
            if (!_gaugeReportersMap.TryGetValue(metricKey, out var gauge)) return;
            var labelValues = _ensureLabels(tags, gauge.LabelNames);
            gauge.WithLabels(labelValues).Set(value);
        }

        public void ReportSummary(string metricKey, Dictionary<string, string> tags, double value)
        {
            if (!_summaryReportersMap.TryGetValue(metricKey, out var summary)) return;
            var labelValues = _ensureLabels(tags, summary.LabelNames);
            summary.WithLabels(labelValues).Observe(value);
        }

        private string[] _ensureLabels(Dictionary<string, string> labels, string[] labelNames)
        {
            var labelValues = new List<string>();
            foreach (var lName in labelNames)
            {
                if (labels.TryGetValue(lName, out var lValue))
                {
                    labelValues.Add(lValue);
                    continue;
                }
                if (_additionalLabels.TryGetValue(lName, out lValue))
                {
                    labelValues.Add(lValue);
                    continue;
                }
                if (_constantLabels.TryGetValue(lName, out lValue))
                {
                    labelValues.Add(lValue);
                    continue;
                }

                labelValues.Add("");
            }

            return labelValues.ToArray();
        }
    }
}