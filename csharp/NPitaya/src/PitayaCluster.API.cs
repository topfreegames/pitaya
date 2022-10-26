using System;
using Google.Protobuf;
using System.Collections.Generic;
using System.Collections.Concurrent;
using System.Diagnostics;
using System.Threading;
using System.Threading.Tasks;
using NPitaya.Metrics;
using NPitaya.Models;
using NPitaya.Serializer;
using NPitaya.Protos;
using NPitaya.Utils;
using static NPitaya.Utils.Utils;
using System.Linq;
using Jaeger;
using Jaeger.Samplers;
using Jaeger.Reporters;
using Jaeger.Senders.Thrift;
using Microsoft.Extensions.DependencyInjection;
using OpenTracing.Util;

// TODO profiling
// TODO better reflection performance in task async call
// TODO support to sync methods
// TODO benchmark with blocking handlers
namespace NPitaya
{
    public partial class PitayaCluster
    {
        private static readonly int ProcessorsCount = Environment.ProcessorCount;
        private static ISerializer _serializer = new ProtobufSerializer();
        public delegate string RemoteNameFunc(string methodName);
        private delegate void OnSignalFunc();
        private static readonly Dictionary<string, RemoteMethod> RemotesDict = new Dictionary<string, RemoteMethod>();
        private static readonly Dictionary<string, RemoteMethod> HandlersDict = new Dictionary<string, RemoteMethod>();
        private static readonly LimitedConcurrencyLevelTaskScheduler Lcts = new LimitedConcurrencyLevelTaskScheduler(ProcessorsCount);
        private static TaskFactory _rpcTaskFactory = new TaskFactory(Lcts);

        private static Action _onSignalEvent;
        private static bool _processedSigint = false;

        private static Action<SDEvent> _onSDEvent;

        // Sidecar stuff
        
        static BlockingCollection<SidecarRequest> queueRead = new BlockingCollection<SidecarRequest>(PitayaConfiguration.Config.getInt(PitayaConfiguration.CONFIG_READBUFFER_SIZE));
        static BlockingCollection<RPCResponse> queueWrite = new BlockingCollection<RPCResponse>(PitayaConfiguration.Config.getInt(PitayaConfiguration.CONFIG_WRITEBUFFER_SIZE));

        // Tracer
        private static Tracer _tracer;
        
        public static void Initialize(string sidecarListenSocket, 
            Server server = null, 
            Action<SDEvent> cbServiceDiscovery = null, 
            IServiceProvider serviceProvider = null, 
            bool debug = false){
            Initialize(sidecarListenSocket, 0, server, cbServiceDiscovery, serviceProvider, debug);
        }

        public static void Initialize(string sidecarAddr, 
            int sidecarPort, 
            Server server = null, 
            Action<SDEvent> cbServiceDiscovery = null, 
            IServiceProvider serviceProvider = null, 
            bool debug = false){
            if (_isInitialized){
                Logger.Warn("Initialize called but pitaya is already initialized");
                return;
            }
            
            if (server != null)
            {
                InitializeSidecarClientWithPitaya(sidecarAddr, sidecarPort, server, debug);
            }
            else
            {
                InitializeSidecarClient(sidecarAddr, sidecarPort);
            }
            
            InitializeInternal(cbServiceDiscovery, serviceProvider);
        }
        
        static void InitializeInternal(Action<SDEvent> cbServiceDiscovery, IServiceProvider serviceProvider){
            if (_client == null)
            {
                throw new PitayaException("Initialization failed");
            }

            RegisterRemotesAndHandlers(serviceProvider);

            ListenToIncomingRPCs(_client);
            SetServiceDiscoveryListener(cbServiceDiscovery);
            ListenSDEvents(_client);
            RegisterGracefulShutdown();
        }
        
        // TODO this should now be a pure csharp implementation of getting sigint/sigterm
        public static void AddSignalHandler(Action cb)
        {
            _onSignalEvent += cb;
        }

        private static void OnSignal()
        {
            Logger.Info("Invoking signal handler");
            _onSignalEvent?.Invoke();
        }

        private static List<Type> GetAllInheriting(Type type)
        {
            return AppDomain.CurrentDomain.GetAssemblies().SelectMany(x => x.GetTypes())
                .Where(x => type.IsAssignableFrom(x) && !x.IsInterface && !x.IsAbstract && x.FullName != type.FullName)
                .Select(x => x).ToList();
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

        private static void RegisterRemotesAndHandlers(IServiceProvider serviceProvider){
            var handlers = GetAllInheriting(typeof(BaseHandler));
            foreach (var handler in handlers){
                RegisterHandler((BaseHandler)ActivatorUtilities.CreateInstance(serviceProvider, handler));
            }

            var remotes = GetAllInheriting(typeof(BaseRemote));
            foreach (var remote in remotes){
                RegisterRemote((BaseRemote)ActivatorUtilities.CreateInstance(serviceProvider, remote));
            }
        }

        public static void StartJaeger(
            Server server,
            String serviceName,
            double probabilisticSamplerParam)
        {
            // TODO understand and change this loggerFactory
            System.Environment.SetEnvironmentVariable("JAEGER_SERVICE_NAME", serviceName);
            System.Environment.SetEnvironmentVariable("JAEGER_SAMPLER_PARAM", probabilisticSamplerParam.ToString());
            if (System.Environment.GetEnvironmentVariable("JAEGER_AGENT_HOST") == null)
            {
                System.Environment.SetEnvironmentVariable("JAEGER_AGENT_HOST", "localhost");
            }
            if (System.Environment.GetEnvironmentVariable("JAEGER_AGENT_PORT") == null)
            {
                System.Environment.SetEnvironmentVariable("JAEGER_AGENT_PORT", "6831");
            }
            var loggingFactory = new Microsoft.Extensions.Logging.LoggerFactory();
            var config = Configuration.FromEnv(loggingFactory);
            var sampler = new ProbabilisticSampler(config.SamplerConfig.Param.Value);
            var reporter = new RemoteReporter.Builder()
                .WithLoggerFactory(loggingFactory)
                .WithSender(new UdpSender(config.ReporterConfig.SenderConfig.AgentHost, config.ReporterConfig.SenderConfig.AgentPort.Value, 0))
                .Build();
            _tracer = new Tracer.Builder(serviceName)
                .WithLoggerFactory(loggingFactory)
                .WithSampler(sampler)
                .WithReporter(reporter)
                .WithTag("local.id", server.Id)
                .WithTag("local.type", server.Type)
                .WithTag("span.kind", "sidecar")
                .Build();
            if (!GlobalTracer.IsRegistered()){
                GlobalTracer.Register(_tracer);
            }
        }

        private static void RegisterGracefulShutdown(){
            Console.CancelKeyPress += (_, ea) =>
            {
                _processedSigint = true;
                Console.WriteLine("Received SIGINT (Ctrl+C), executing on signal function");
                OnSignal();
                Terminate();
            };

            AppDomain.CurrentDomain.ProcessExit += (_, ea) =>
            {
                if (_processedSigint) {
                    Console.WriteLine("Ignoring SIGTERM, already processed SIGINT");
                } else{
                    Console.WriteLine("Received SIGTERM, executing on signal function");
                    OnSignal();
                    Terminate();
                }
            };
        }

        private static void RegisterRemote(BaseRemote remote)
        {
            string className = DefaultRemoteNameFunc(remote.GetName());
            RegisterRemote(remote, className, DefaultRemoteNameFunc);
        }

        private static void RegisterRemote(BaseRemote remote, string name, RemoteNameFunc remoteNameFunc)
        {
            Dictionary<string, RemoteMethod> m = remote.GetRemotesMap();
            foreach (KeyValuePair<string, RemoteMethod> kvp in m)
            {
                var rn = remoteNameFunc(kvp.Key);
                var remoteName = $"{name}.{rn}";
                if (RemotesDict.ContainsKey(remoteName))
                {
                    throw new PitayaException($"tried to register same remote twice! remote name: {remoteName}");
                }

                Logger.Info("registering remote {0}", remoteName);
                RemotesDict[remoteName] = kvp.Value;
            }
        }

        private static void RegisterHandler(BaseHandler handler)
        {
            string className = DefaultRemoteNameFunc(handler.GetName());
            RegisterHandler(handler, className, DefaultRemoteNameFunc);
        }

        public static void RegisterHandler(BaseHandler handler, string name, RemoteNameFunc remoteNameFunc)
        {
            Dictionary<string, RemoteMethod> m = handler.GetRemotesMap();
            foreach (KeyValuePair<string, RemoteMethod> kvp in m)
            {
                var rn = remoteNameFunc(kvp.Key);
                var handlerName = $"{name}.{rn}";
                if (HandlersDict.ContainsKey(handlerName))
                {
                    throw new PitayaException($"tried to register same remote twice! remote name: {handlerName}");
                }

                Logger.Info("registering handler {0}", handlerName);
                HandlersDict[handlerName] = kvp.Value;
            }
        }

        public static void SetSerializer(ISerializer s)
        {
            _serializer = s;
        }
        public static void Terminate()
        {
            if (_isInitialized){
                UnsetServiceDiscoveryListener();
                _client.StopPitaya(new Google.Protobuf.WellKnownTypes.Empty());
                ShutdownSidecar();
            }
        }

        public static async void TerminateAsync()
        {
            if (_isInitialized){
                UnsetServiceDiscoveryListener();
                await _client.StopPitayaAsync(new Google.Protobuf.WellKnownTypes.Empty());
                await ShutdownSidecarAsync();
            }
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

        // TODO deprecate this frontendId field, it's not useful when using nats
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

        private static void SetServiceDiscoveryListener(Action<SDEvent> cb)
        {
            _onSDEvent += cb;
        }

        private static void UnsetServiceDiscoveryListener()
        {
            if (_onSDEvent != null)
            {
                _onSDEvent = null;
            }
        }
    }
}
