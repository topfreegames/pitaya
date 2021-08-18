using System;
using System.Collections.Generic;
using System.Threading;
using NPitaya;
using NPitaya.Metrics;
using NPitaya.Models;

namespace PitayaCSharpExample
{
  class Example
  {
    static void Main(string[] args)
    {
      Logger.SetLevel(LogLevel.DEBUG);

      Console.WriteLine("c# prog running");

      string serverId = System.Guid.NewGuid().ToString();

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

      var metadata = new Dictionary<string, string>(){
        {"testKey", "testValue"}
      };

      var sv = new NPitaya.Protos.Server{ 
        Id = serverId,
        Type = "csharp",
        Hostname = "localhost",
        Frontend = false };
      sv.Metadata.Add(metadata);

      var natsConfig = new NatsConfig(
          endpoint: "127.0.0.1:4222",
          connectionTimeoutMs: 2000,
          requestTimeoutMs: 1000,
          serverShutdownDeadlineMs: 3,
          serverMaxNumberOfRpcs: 100,
          maxConnectionRetries: 3,
          maxPendingMessages: 1000);

      var grpcConfig = new GrpcConfig(
        host: "127.0.0.1",
        port: 5444,
        serverShutdownDeadlineMs: 2000,
        serverMaxNumberOfRpcs: 200,
        clientRpcTimeoutMs: 10000
      );

      Dictionary<string, string> constantTags = new Dictionary<string, string>
      {
          {"game", "game"},
          {"serverType", "svType"}
      };
      var statsdMR = new StatsdMetricsReporter("localhost", 5000, "game", constantTags);
      MetricsReporters.AddMetricReporter(statsdMR);
      var prometheusMR = new PrometheusMetricsReporter("default", "game", 9090);
      MetricsReporters.AddMetricReporter(prometheusMR);

      PitayaCluster.AddSignalHandler(() =>
      {
        Logger.Info("Signal received, exiting app");
        Environment.Exit(1);
        //Environment.FailFast("oops");
      });

      try
      {
        PitayaCluster.SetSerializer(new NPitaya.Serializer.JSONSerializer());
        var sockAddr = "unix://" + System.IO.Path.Combine(System.IO.Path.GetTempPath(), "pitaya.sock");
        PitayaCluster.Initialize(
            sockAddr,
            sv,
            true,
            (sdEvent) => {
              switch (sdEvent.Event)
              {
                case NPitaya.Protos.SDEvent.Types.Event.Add:
                  Console.WriteLine("Server was added");
                  Console.WriteLine("   id: " + sdEvent.Server.Id);
                  Console.WriteLine(" type: " + sdEvent.Server.Type);
                  break;
                case NPitaya.Protos.SDEvent.Types.Event.Remove:
                  Console.WriteLine("Server was removed");
                  Console.WriteLine("   id: " + sdEvent.Server.Id);
                  Console.WriteLine(" type: " + sdEvent.Server.Type);
                  break;
                default:
                  throw new ArgumentOutOfRangeException(nameof(sdEvent.Event), sdEvent.Event, null);
              }
            }
          );
      }
      catch (PitayaException exc)
      {
        Logger.Error("Failed to create cluster: {0}", exc.Message);
        Environment.Exit(1);
      }

      Logger.Info("pitaya lib initialized successfully :)");

      Thread.Sleep(1000);

      //TrySendRpc();
      Console.ReadKey();
      //PitayaCluster.Terminate();
    }

    static async void TrySendRpc(){
        try
        {
            var res = await PitayaCluster.Rpc<Protos.RPCRes>(Route.FromString("csharp.testRemote.remote"),
                null);
            Console.WriteLine($"Code: {res.Code}");
            Console.WriteLine($"Msg: {res.Msg}");
        }
        catch (PitayaException e)
        {
            Logger.Error("Error sending RPC Call: {0}", e.Message);
        }
    }
    // TODO now I need to create a handler in this example so that I call a route and then send a push from there
    // Confirm the RPC sending capabilities are working
  }
}