using System.Collections.Generic;
using System.Diagnostics;
using System.Threading;
using NPitaya.Metrics;
using Xunit;

namespace NPitayaTest.Tests.Metric
{
    public class Prometheus
    {
        [Fact]
        public void Can_Report_Pitaya_Metrics()
        {
            var cm = new CustomMetricsSpec();
            cm.Counters.Add(new CustomCounter() { Subsystem = "subsystem", Name = "nameC", Help = "help"});
            cm.Gauges.Add(new CustomGauge() { Subsystem = "subsystem", Name = "nameG", Help = "help"});
            cm.Summaries.Add(new CustomSummary() { Subsystem = "subsystem", Name = "nameS", Help = "help", Objectives = null});
            var prometheusMR = new PrometheusMetricsReporter("default", "game", 9090, null, null, cm);
            MetricsReporters.AddMetricReporter(prometheusMR);

            for (var i = 0; i < 5; ++i)
            {
                new Thread(() =>
                {
                    Thread.CurrentThread.IsBackground = true;

                    MetricsReporters.ReportTimer("success", "game.remoteGame.test", "rpc", "", Stopwatch.StartNew());
                    MetricsReporters.ReportMessageProccessDelay("game.remoteGame.test", "rpc", Stopwatch.StartNew());
                    MetricsReporters.ReportNumberOfConnectedClients(1);
                    MetricsReporters.ReportSummary("nameS", new Dictionary<string, string>(), 1);
                    MetricsReporters.ReportGauge("nameG", new Dictionary<string, string>(), 1);
                    MetricsReporters.ReportCount("nameC", new Dictionary<string, string>(), 1);
                }).Start();
            }
        }
    }
}