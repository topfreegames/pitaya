using System.Collections.Generic;
using System.Diagnostics;

namespace NPitaya.Metrics
{
    public static class MetricsReporters
    {
        private static List<IMetricsReporter> _reporters = new List<IMetricsReporter>();

        public static void AddMetricReporter(IMetricsReporter mr)
        {
            _reporters.Add(mr);
        }

        public static void Terminate()
        {
            _reporters = new List<IMetricsReporter>();
        }

        public static void ReportMessageProccessDelay(string route, string type, Stopwatch sw)
        {
            var tags = new Dictionary<string, string> { {"route", route}, {"type", type} };
            ReportSummary(Constants.ProccessDelayMetricKey, tags, sw.Elapsed.TotalMilliseconds * 1000000);
        }

        public static void ReportTimer(string status, string route, string type, string code, Stopwatch sw)
        {
            var tags = new Dictionary<string, string> { {"status", status}, {"route", route}, {"type", type}, {"code", code} };
            ReportSummary(Constants.ResponseTimeMetricKey, tags, sw.Elapsed.TotalMilliseconds * 1000000);
        }

        public static void ReportNumberOfConnectedClients(double value)
        {
            ReportGauge(Constants.ConnectedClientsMetricKey, new Dictionary<string, string>(), value);
        }

        public static void ReportCount(string key, Dictionary<string, string> tags, double value)
        {
            foreach (var reporter in _reporters)
            {
                reporter.ReportCount(key, tags, value);
            }
        }
        public static void ReportGauge(string key, Dictionary<string, string> tags, double value)
        {
            foreach (var reporter in _reporters)
            {
                reporter.ReportGauge(key, tags, value);
            }
        }

        public static void ReportSummary(string key, Dictionary<string, string> tags, double value)
        {
            foreach (var reporter in _reporters)
            {
                reporter.ReportSummary(key, tags, value);
            }
        }

    }
}