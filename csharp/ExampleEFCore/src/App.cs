using System;
using System.Collections.Generic;
using System.Threading;
using ExampleORM.Servers.BusinessLogic.Handlers;
using NPitaya;
using NPitaya.Models;
using NPitaya.Serializer;

namespace ExampleORM
{
    class App
    {
        private static void Main(string[] args)
        {
            Logger.SetLevel(LogLevel.INFO);

            var serverId = System.Guid.NewGuid().ToString();

            var sdConfig = new SDConfig(
                endpoints: "http://127.0.0.1:2379",
                etcdPrefix: "pitaya/",
                serverTypeFilters: new List<string>(),
                heartbeatTTLSec: 60,
                logHeartbeat: true,
                logServerSync: true,
                logServerDetails: true,
                syncServersIntervalSec: 30,
                maxNumberOfRetries: 0);

            var sv = new Server(
                id: serverId,
                type: "csharp",
                metadata: "",
                hostname: "localhost",
                frontend: false);

            var grpcConfig = new GrpcConfig(
                host: "127.0.0.1",
                port: 5444,
                serverShutdownDeadlineMs: 3000,
                serverMaxNumberOfRpcs: 500,
                clientRpcTimeoutMs: 10000
            );

            PitayaCluster.AddSignalHandler(() =>
            {
                Logger.Info("Calling terminate on cluster");
                PitayaCluster.Terminate();
                Logger.Info("Cluster terminated, exiting app");
                Environment.Exit(1);
            });

            PitayaCluster.SetSerializer(new JSONSerializer()); // Using json serializer for easier interop with pitaya-cli

            try
            {
                PitayaCluster.Initialize(
                    grpcConfig,
                    sdConfig,
                    sv,
                    NativeLogLevel.Debug,
                    new PitayaCluster.ServiceDiscoveryListener((action, server) =>
                    {
                        switch (action)
                        {
                            case PitayaCluster.ServiceDiscoveryAction.ServerAdded:
                                Console.WriteLine($"Server added: {server}");
                                break;
                            case PitayaCluster.ServiceDiscoveryAction.ServerRemoved:
                                Console.WriteLine($"Server removed: {server}");
                                break;
                            default:
                                throw new ArgumentOutOfRangeException(nameof(action), action, null);
                        }
                    })
                );
            }
            catch (PitayaException exc)
            {
                Logger.Error("Failed to create cluster: {0}", exc.Message);
                Environment.Exit(1);
            }

            while (true)
            {
                Thread.Sleep(10);
            }
            ////
        }
    }
}
