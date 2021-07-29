using System;
using Grpc.Core;
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

            // TODO see if I can get an error here, or else kill this process when the channel dies,
            // because then probably the sidecar died
            client.StartPitaya(req);
            return client;
        }
    }
}
