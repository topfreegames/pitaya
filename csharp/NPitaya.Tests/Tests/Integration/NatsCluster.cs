using System;
using System.Collections.Generic;
using System.Threading;
using NPitaya;
using NPitaya.Metrics;
using Xunit;

namespace NPitayaTest.Tests.Integration
{
    public class NatsCluster
    {
        [Fact]
        public void Can_Fail_Initialization_Multiple_Times()
        {
            var natsConfig = new NatsConfig(
                endpoint: "http://127.0.0.1:4222",
                connectionTimeoutMs: 1000,
                requestTimeoutMs: 5000,
                serverShutdownDeadlineMs: 10 * 1000,
                serverMaxNumberOfRpcs: int.MaxValue,
                maxConnectionRetries: 3,
                maxPendingMessages: 100);

            var sdCfg = new SDConfig(
                endpoints: "127.0.0.1:123123123",
                etcdPrefix: "pitaya/",
                serverTypeFilters: new List<string>(),
                heartbeatTTLSec: 10,
                logHeartbeat: false,
                logServerSync: false,
                logServerDetails: false,
                syncServersIntervalSec: 10,
                maxNumberOfRetries: 0
            );

            var server = new Server(
                id: "id",
                type: "type",
                metadata: "",
                hostname: "",
                frontend: false
            );

            for (var i = 0; i < 10; ++i)
            {
                Assert.Throws<PitayaException>(() =>
                {
                    PitayaCluster.Initialize(natsConfig, sdCfg, server, NativeLogLevel.Debug);
                });
            }

            PitayaCluster.Terminate();
        }

        [Fact]
        public void Can_Be_Initialized_And_Terminated_Multiple_Times()
        {
            var natsConfig = new NatsConfig(
                endpoint: "http://127.0.0.1:4222",
                connectionTimeoutMs: 1000,
                requestTimeoutMs: 5000,
                serverShutdownDeadlineMs: 10 * 1000,
                serverMaxNumberOfRpcs: int.MaxValue,
                maxConnectionRetries: 3,
                maxPendingMessages: 100);

            var sdCfg = new SDConfig(
                endpoints: "http://127.0.0.1:2379",
                etcdPrefix: "pitaya/",
                serverTypeFilters: new List<string>(),
                heartbeatTTLSec: 10,
                logHeartbeat: false,
                logServerSync: false,
                logServerDetails: false,
                syncServersIntervalSec: 10,
                maxNumberOfRetries: 0
            );

            var server = new Server(
                id: "myserverid",
                type: "myservertype",
                metadata: "",
                hostname: "",
                frontend: false
            );

            for (var i = 0; i < 2; ++i)
            {
                PitayaCluster.Initialize(natsConfig, sdCfg, server, NativeLogLevel.Debug);
                PitayaCluster.Terminate();
            }
        }

    }
}