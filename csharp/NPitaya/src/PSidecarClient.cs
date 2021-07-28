using System;
using Grpc.Core;
using NPitaya.Protos;
using System.Net.Http;
using System.Net;
using System.Net.Sockets;
using Grpc.Net.Client;

namespace NPitaya
{

    public class UnixDomainSocketConnectionFactory
    {
        private readonly EndPoint _endPoint;
        public UnixDomainSocketConnectionFactory(EndPoint endPoint)
        {
            _endPoint = endPoint;
        }
        public async System.Threading.Tasks.ValueTask<System.IO.Stream> ConnectAsync(SocketsHttpConnectionContext _,
            System.Threading.CancellationToken cancellationToken = default)
        {
            var socket = new Socket(AddressFamily.Unix, SocketType.Stream, ProtocolType.Unspecified);
            try
            {
                await socket.ConnectAsync(_endPoint, cancellationToken).ConfigureAwait(false);
                return new NetworkStream(socket, true);
            }
            catch
            {
                socket.Dispose();
                throw;
            }
        }
    }
    public partial class PitayaCluster
    {
        public static Sidecar.SidecarClient InitializeSidecarClient(string sidecarListenAddr, NPitaya.Protos.Server server, bool debug = false)
        {
            GrpcEnvironment.SetCompletionQueueCount(Environment.ProcessorCount);
            GrpcEnvironment.SetThreadPoolSize(Environment.ProcessorCount);
            // TODO tls stuff

            var udsEndPoint = new UnixDomainSocketEndPoint(sidecarListenAddr);
            var connectionFactory = new UnixDomainSocketConnectionFactory(udsEndPoint);
            var socketsHttpHandler = new SocketsHttpHandler
            {
                ConnectCallback = connectionFactory.ConnectAsync
            };

            var channel = GrpcChannel.ForAddress("http://localhost", new GrpcChannelOptions
            {
                HttpHandler = socketsHttpHandler
            });


            var client = new Sidecar.SidecarClient(channel);
            var req = new StartPitayaRequest { Config = server, DebugLog = debug };

            // TODO see if I can get an error here, or else kill this process when the channel dies,
            // because then probably the sidecar died
            client.StartPitaya(req);
            return client;
        }
    }
}
