using System;
using System.Collections.Generic;
using System.Threading;
using NPitaya;
using NPitaya.Metrics;
using NPitaya.Models;
using NPitaya.Protos;

namespace PitayaCSharpExample
{
  class Example
  {
    static void Main(string[] args)
    {
      Logger.SetLevel(LogLevel.DEBUG);

      Logger.Info("Starting Pitaya C#");

      string serverId = System.Guid.NewGuid().ToString();

      var metadata = new Dictionary<string, string>(){
        {"testKey", "testValue"}
      };

      var sv = new NPitaya.Protos.Server{ 
        Id = serverId,
        Type = "csharp",
        Hostname = "localhost",
        Frontend = false };
      sv.Metadata.Add(metadata);

      Dictionary<string, string> constantTags = new Dictionary<string, string>
      {
          {"game", "game"},
          {"serverType", "svType"}
      };
      var statsdMR = new StatsdMetricsReporter("localhost", 5000, "game", constantTags);
      MetricsReporters.AddMetricReporter(statsdMR);
      //var prometheusMR = new PrometheusMetricsReporter("default", "game", 9090);
      //MetricsReporters.AddMetricReporter(prometheusMR);

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
        Logger.Info("Connecting to pitaya sidecar at addr: {0}", sockAddr);
        PitayaCluster.StartJaeger(sv, "pitaya-csharp-example", 1.0f);
        PitayaCluster.Initialize(
          sockAddr,
          sv,
          true,
          OnServiceDiscoveryEvent
        );
      }
      catch (PitayaException exc)
      {
        Logger.Error("Failed to create cluster: {0}", exc.Message);
        Environment.Exit(1);
      }

      Logger.Info("Pitaya C# initialized!");

      Thread.Sleep(1000);

      //TrySendRpc();
      Console.ReadKey();
      //PitayaCluster.Terminate();
    }

    static void OnServiceDiscoveryEvent(SDEvent sdEvent){
      switch (sdEvent.Event)
      {
        case NPitaya.Protos.SDEvent.Types.Event.Add:
          Logger.Info($"Server Added - Id: {sdEvent.Server.Id} Type: {sdEvent.Server.Type}");
          break;
        case NPitaya.Protos.SDEvent.Types.Event.Remove:
          Logger.Info($"Server Removed - Id: {sdEvent.Server.Id} Type: {sdEvent.Server.Type}");
          break;
        default:
          throw new ArgumentOutOfRangeException(nameof(sdEvent.Event), sdEvent.Event, null);
      }
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
    // Confirm the RPC sending capabilities are working
  }
}
