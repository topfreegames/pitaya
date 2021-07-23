using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using PitayaSimpleJson;
using Grpc.Core;
using NPitaya.Protos;
using System.Threading;
using System.Threading.Tasks;
using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using System.Collections.Concurrent;

namespace NPitaya
{
    public partial class PitayaCluster
    {
        public static Sidecar.SidecarClient InitializeSidecarClient(string sidecarListenAddr, Server server, bool debug = false)
        {
            GrpcEnvironment.SetCompletionQueueCount(Environment.ProcessorCount);
            GrpcEnvironment.SetThreadPoolSize(Environment.ProcessorCount);
            // TODO tls stuff
            Channel channel = new Channel(sidecarListenAddr, ChannelCredentials.Insecure);

            var client = new Sidecar.SidecarClient(channel);
            var serverConfig = new StartPitayaRequest.Types.ServerConfig { IsFrontend = server.frontend, ServerType = server.type, DebugLog = debug};
            serverConfig.Metadata.Add(server.metadata);
            var req = new StartPitayaRequest { Config = serverConfig };

            // TODO see if I can get an error here, or else kill this process when the channel dies,
            // because then probably the sidecar died
            client.StartPitaya(req);
            return client;
        }
    }
}
