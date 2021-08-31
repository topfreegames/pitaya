using System;
using Grpc.Core;
using NPitaya.Models;
using NPitaya.Protos;

namespace NPitaya
{

    public partial class PitayaCluster
    {
        public static Sidecar.SidecarClient InitializeSidecarClient(string sidecarListenAddr, NPitaya.Protos.Server server, bool debug = false)
        {
            GrpcEnvironment.SetCompletionQueueCount(Environment.ProcessorCount);
            GrpcEnvironment.SetThreadPoolSize(Environment.ProcessorCount);
            // TODO tls stuff

            Channel channel = new Channel(sidecarListenAddr, ChannelCredentials.Insecure);

            var client = new Sidecar.SidecarClient(channel);
            var req = new StartPitayaRequest { Config = server, DebugLog = debug };

            client.StartPitaya(req);
            // this is a hacky approach to detect if server is not running anymore, and if not, die
            new System.Threading.Thread(() =>
            {
                for(;;)
                {
                    client.Heartbeat(new Google.Protobuf.WellKnownTypes.Empty());
                    var timeoutMs = PitayaConfiguration.Config.getInt(PitayaConfiguration.CONFIG_HEARTBEAT_TIMEOUT_MS);
                    System.Threading.Thread.Sleep(timeoutMs);
                }
            }).Start();
            return client;
        }
    }
}
