using System.Collections.Generic;
using NPitaya;
using Xunit;
[assembly: CollectionBehavior(DisableTestParallelization = true)]

namespace NPitayaTest.Tests.Integration
{
    public class GrpcCluster
    {
        [Fact]
        public void Can_Fail_Initialization_Multiple_Times()
        {
            var grpcConfig = new GrpcConfig(
                host: "127.0.0.1",
                port: 40405,
                serverShutdownDeadlineMs: 2000,
                serverMaxNumberOfRpcs: 100,
                clientRpcTimeoutMs: 10000
            );

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
                    PitayaCluster.Initialize(grpcConfig, sdCfg, server, NativeLogLevel.Debug);
                });
            }

            PitayaCluster.Terminate();
        }

        [Fact]
        public void Can_Be_Initialized_And_Terminated_Multiple_Times()
        {
            var grpcConfig = new GrpcConfig(
                host: "127.0.0.1",
                port: 40405,
                serverShutdownDeadlineMs: 2000,
                serverMaxNumberOfRpcs: 100,
                clientRpcTimeoutMs: 10000
            );

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
                PitayaCluster.Initialize(grpcConfig, sdCfg, server, NativeLogLevel.Debug);
                PitayaCluster.Terminate();
            }
        }
    }
}