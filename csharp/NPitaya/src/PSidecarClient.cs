using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Diagnostics;
using System.Threading;
using Grpc.Core;
using NPitaya.Models;
using NPitaya.Protos;
using System.Threading.Tasks;
using Google.Protobuf;
using NPitaya.Metrics;
using NPitaya.Serializer;
using static NPitaya.Utils.Utils;
using Server = NPitaya.Protos.Server;

namespace NPitaya
{
    public partial class PitayaCluster
    {
        private static bool _isInitialized = false;
        private static Sidecar.SidecarClient _client = null;
        private static Grpc.Core.Channel _channel;

        static BlockingCollection<SidecarRequest> queueRead = new BlockingCollection<SidecarRequest>(PitayaConfiguration.Config.getInt(PitayaConfiguration.CONFIG_READBUFFER_SIZE));
        static BlockingCollection<RPCResponse> queueWrite = new BlockingCollection<RPCResponse>(PitayaConfiguration.Config.getInt(PitayaConfiguration.CONFIG_WRITEBUFFER_SIZE));

        static PitayaCluster(){
            GrpcEnvironment.SetCompletionQueueCount(Environment.ProcessorCount);
            GrpcEnvironment.SetThreadPoolSize(Environment.ProcessorCount);
        }

        public static void InitializeSidecarClientWithPitaya(string sidecarAddr, int sidecarPort, NPitaya.Protos.Server server, bool debug = false)
        {
            if (_isInitialized)
            {
                Logger.Warn("Pitaya is already initialized!");
                return;
            }

           InitializeSidecarClient(sidecarAddr, sidecarPort);
           InitializePitaya(server, debug);
        }

        public static void InitializeSidecarClient(string sidecarAddr, int sidecarPort)
        {
            if (_isInitialized)
            {
                Logger.Warn("Pitaya is already initialized!");
                return;
            }

            if (sidecarPort != 0)
            {
                _channel = new Channel(sidecarAddr, sidecarPort, ChannelCredentials.Insecure);
            }
            else
            {
                _channel = new Channel(sidecarAddr, ChannelCredentials.Insecure);
            }

            _client = new Sidecar.SidecarClient(_channel);
            
            // this is a hacky approach to detect if server is not running anymore, and if not, die
            new System.Threading.Thread(async() =>
            {
                while (!_channel.ShutdownToken.IsCancellationRequested)
                {
                    _client.Heartbeat(new Google.Protobuf.WellKnownTypes.Empty());
                    var timeoutMs = PitayaConfiguration.Config.getInt(PitayaConfiguration.CONFIG_HEARTBEAT_TIMEOUT_MS);
                    await Task.Delay(TimeSpan.FromMilliseconds(timeoutMs), 
                        _channel.ShutdownToken);
                }
            }).Start();
            
            ListenToIncomingRPCs(_client);
            ListenSDEvents(_client);
            
            _isInitialized = true;
        }
        static void InitializePitaya(NPitaya.Protos.Server server, bool debug = false){
            var req = new StartPitayaRequest { Config = server, DebugLog = debug };
            _client.StartPitaya(req);
        }
        
        private static void ListenToIncomingRPCs(Sidecar.SidecarClient client)
        {
            var stream = client.ListenRPC();

            new Thread(async () =>{
                while (await stream.ResponseStream.MoveNext(_channel.ShutdownToken))
                {
                    var current = stream.ResponseStream.Current;
                    queueRead.Add(current);
                }
            }).Start();

            new Thread(async () =>{
                while (true)
                {
                    var res = queueWrite.Take();
                    await stream.RequestStream.WriteAsync(res);
                }
            }).Start();

            for (int i = 0; i < ProcessorsCount; i++)
            {
                var threadId = i + 1;
                new Thread(async () =>
                {
                    Logger.Debug($"[Consumer thread {threadId}] Started");
                    while (!_channel.ShutdownToken.IsCancellationRequested)
                    {
                        var req = queueRead.Take(_channel.ShutdownToken);
                        //#pragma warning disable 4014
                        var res = await HandleIncomingRpc(req.Req);
                        //#pragma warning restore 4014
                        queueWrite.Add(new Protos.RPCResponse { ReqId = req.ReqId, Res = res });
                    }
                }).Start();
            }
        }

        private static void ListenSDEvents(Sidecar.SidecarClient client)
        {
            var stream = client.ListenSD(new Google.Protobuf.WellKnownTypes.Empty());
            new Thread(async () =>{
                while (await stream.ResponseStream.MoveNext(_channel.ShutdownToken))
                {
                    var current = stream.ResponseStream.Current;
                    if (_onSDEvent != null){
                        _onSDEvent(current);
                    }
                }
            }).Start();
        }
        
        public static Task<PushResponse> SendPushToUser(string serverType, string route, string uid,
            object pushMsg)
        {
            // TODO see if this taskfactory is still required
            return _rpcTaskFactory.StartNew(() =>
            {
                var push = new Push
                {
                    Route = route,
                    Uid = uid,
                    Data = ByteString.CopyFrom(SerializerUtils.SerializeOrRaw(pushMsg, _serializer))
                };
                
                var span = _tracer.BuildSpan("system.push")
                    .WithTag("peer.serverType", serverType)
                    .Start();
                var res = _client.SendPush(new PushRequest{FrontendType=serverType, Push=push}, GRPCMetadataWithSpanContext(span));
                span.Finish();
                return res;
            });
        }

        public static Task<PushResponse> SendKickToUser(string serverType, KickMsg kick)
        {
            return _rpcTaskFactory.StartNew(() =>
            {
                var span = _tracer.BuildSpan("system.kick")
                    .WithTag("peer.serverType", serverType)
                    .Start();
                var res = _client.SendKick(new KickRequest{FrontendType=serverType, Kick=kick}, GRPCMetadataWithSpanContext(span));
                span.Finish();
                return res;
            });
        }
        
        public static Task<T> Rpc<T>(string serverId, Route route, object msg)
        {
            return _rpcTaskFactory.StartNew(() =>
            {
                var retError = new Error();
                var ok = false;
                Response res = null;
                Stopwatch sw = null;
                var span = _tracer.BuildSpan(route.ToString())
                    .WithTag("peer.id", serverId)
                    .WithTag("peer.serverType", route.svType)
                    .Start();
                try
                {
                    var data = SerializerUtils.SerializeOrRaw(msg, _serializer);
                    sw = Stopwatch.StartNew();
                    // TODO this can be optimized I think by using a readonly span
                    res = _client.SendRPC(new RequestTo{ServerID=serverId, Msg=new Msg{Route=route.ToString(), Data=ByteString.CopyFrom(data.AsSpan()), Type=MsgType.MsgRequest}}, GRPCMetadataWithSpanContext(span));
                    sw.Stop();
                    var protoRet = GetProtoMessageFromResponse<T>(res);
                    return protoRet;
                }
                finally
                {
                    if (sw != null)
                    {
                        if (ok)
                        {
                            MetricsReporters.ReportTimer(Metrics.Constants.Status.success.ToString(), route.ToString(),
                                "rpc", "", sw);
                        }
                        else
                        {
                            MetricsReporters.ReportTimer(Metrics.Constants.Status.fail.ToString(), route.ToString(),
                                "rpc", $"{retError.code}", sw);
                        }
                    }
                    span.Finish();
                }
            });
        }

        public static Task<T> Rpc<T>(Route route, object msg)
        {
            return Rpc<T>("", route, msg);
        }
        
        public static Server GetServerById(string serverId)
        {
            try{
                var protoSv = _client.GetServer(new NPitaya.Protos.Server{Id=serverId});
                return protoSv;
            } catch (Exception){
                return null;
            }
        }

        // TODO find better place for this method
        private static Grpc.Core.Metadata GRPCMetadataWithSpanContext(OpenTracing.ISpan span){
            var dictionary = new Dictionary<string, string>();
            _tracer.Inject(span.Context, OpenTracing.Propagation.BuiltinFormats.HttpHeaders,
                new OpenTracing.Propagation.TextMapInjectAdapter(dictionary));
            Grpc.Core.Metadata metadata = new Grpc.Core.Metadata();
            foreach (var kvp in dictionary)
            {
                metadata.Add(kvp.Key, kvp.Value);
            }
            return metadata;
        }
        
        public static void ShutdownSidecar(){
            if (!_isInitialized){
                return;
            }
            var task = _channel.ShutdownAsync();
            task.Wait();
            _client = null;
            _channel = null;
            _isInitialized = false;
        }
        public static async Task<bool> ShutdownSidecarAsync()
        {
            if (!_isInitialized) {
                return false;
            }
            await _channel.ShutdownAsync();
            _channel = null;
            _client = null;
            _isInitialized = false;
            return true;
        }
    }
}
