using System.Collections.Generic;
using StatsdClient;

namespace NPitaya.Metrics
{
    public class StatsdMetricsReporter: IMetricsReporter
    {
        public StatsdMetricsReporter(string statsdHost, int statsdPort, string metricsPrefix, Dictionary<string, string> constantTags = null)
        {
            var parsedTags = _dictTagsToStringSlice(constantTags);
            StatsdConfig statsdConfig = new StatsdConfig
            {
                StatsdServerName = statsdHost,
                StatsdPort = statsdPort,
                Prefix = metricsPrefix.TrimEnd('.'),
                ConstantTags = parsedTags
            };
            DogStatsd.Configure(statsdConfig);
        }

        private static string[] _dictTagsToStringSlice(Dictionary<string, string> dictTags)
        {
            if (dictTags is null)
            {
                return new string[0];
            }
            var res = new string[dictTags.Count];
            var i = 0;
            foreach (KeyValuePair<string,string> kv in dictTags)
            {
                res[i++] = $"{kv.Key}:{kv.Value}";
            }

            return res;
        }

        public void ReportCount(string metricKey, Dictionary<string, string> tags, double value)
        {
            var parsedTags = _dictTagsToStringSlice(tags);
            DogStatsd.Counter(metricKey, value, tags:parsedTags);
        }

        public void ReportGauge(string metricKey, Dictionary<string, string> tags, double value)
        {
            var parsedTags = _dictTagsToStringSlice(tags);
            DogStatsd.Gauge(metricKey, value, tags:parsedTags);
        }
        public void ReportSummary(string metricKey, Dictionary<string, string> tags, double value)
        {
            var parsedTags = _dictTagsToStringSlice(tags);
            DogStatsd.Timer(metricKey, value, tags:parsedTags);
        }
    }
}