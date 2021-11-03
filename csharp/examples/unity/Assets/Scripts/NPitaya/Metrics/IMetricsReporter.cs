using System.Collections.Generic;

namespace NPitaya.Metrics
{
    public interface IMetricsReporter
    {
        void ReportCount(string metricKey, Dictionary<string, string> labels, double value);
        void ReportGauge(string metricKey, Dictionary<string, string> labels, double value);
        void ReportSummary(string metricKey, Dictionary<string, string> labels, double value);
    }
}