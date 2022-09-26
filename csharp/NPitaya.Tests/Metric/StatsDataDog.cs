using System.Collections.Generic;
using System.Diagnostics;
using System.Threading;
using NPitaya.Metrics;
using Xunit;

namespace NPitayaTest.Tests.Metric
{
    public class StatsDataDog
    {
        [Fact]
        public void Can_Report_Pitaya_Metrics()
        {
            var statsdMR = new StatsdMetricsReporter("localhost", 5000, "game");
            MetricsReporters.AddMetricReporter(statsdMR);

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