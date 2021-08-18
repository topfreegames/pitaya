using System;
using Google.Protobuf;
using System.Collections.Generic;
using System.Collections.Concurrent;
using System.Diagnostics;
using System.Runtime.InteropServices;
using System.Threading;
using System.Threading.Tasks;
using NPitaya.Metrics;
using NPitaya.Models;
using NPitaya.Serializer;
using NPitaya.Protos;
using NPitaya.Utils;
using static NPitaya.Utils.Utils;
using System.Linq;

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
        private static Sidecar.SidecarClient _client = null;
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
        // TODO create configuration on buffer sizes
        
        static BlockingCollection<SidecarRequest> queueRead = new BlockingCollection<SidecarRequest>(1000);
        static BlockingCollection<RPCResponse> queueWrite = new BlockingCollection<RPCResponse>(1000);

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
                while (await stream.ResponseStream.MoveNext(CancellationToken.None))
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
                // TODO see what I can do with this cancellation token
                while (await stream.ResponseStream.MoveNext(CancellationToken.None))
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
                    for (;;) // TODO must exit somehow and log that the thread is exiting
//                  Logger.Debug($"[Consumer thread {threadId}] No more incoming RPCs, exiting");
                    {
                        var req = queueRead.Take();

                        //#pragma warning disable 4014
                        var res = await HandleIncomingRpc(req.Req);
                        //#pragma warning restore 4014
                        queueWrite.Add(new Protos.RPCResponse { ReqId = req.ReqId, Res = res });
                    }
                }).Start();
            }
        }

        private static void RegisterRemotesAndHandlers(){
            var handlers = GetAllInheriting(typeof(BaseHandler));
            foreach (var handler in handlers){
                RegisterHandler((BaseHandler)Activator.CreateInstance(handler));
            }

            var remotes = GetAllInheriting(typeof(BaseRemote));
            foreach (var remote in remotes){
                RegisterRemote((BaseRemote)Activator.CreateInstance(remote));
            }
        }

        public static void Initialize(
                string sidecarListenAddr,
                Server server,
                bool debug,
                Action<SDEvent> cbServiceDiscovery = null
                )
        {
            _client = InitializeSidecarClient(sidecarListenAddr, server, debug);

            if (_client == null)
            {
                throw new PitayaException("Initialization failed");
            }

            RegisterRemotesAndHandlers();

            ListenToIncomingRPCs(_client);
            SetServiceDiscoveryListener(cbServiceDiscovery);
            ListenSDEvents(_client);
            RegisterGracefulShutdown();
        }

        private static void RegisterGracefulShutdown(){
            Console.CancelKeyPress += (_, ea) =>
            {
                _processedSigint = true;
                Console.WriteLine("Received SIGINT (Ctrl+C), executing on signal function");
                OnSignal();
                _client.StopPitaya(new Google.Protobuf.WellKnownTypes.Empty());
            };

            AppDomain.CurrentDomain.ProcessExit += (_, ea) =>
            {
                if (_processedSigint) {
                    Console.WriteLine("Ignoring SIGTERM, already processed SIGINT");
                } else{
                    Console.WriteLine("Received SIGTERM, executing on signal function");
                    OnSignal();
                    _client.StopPitaya(new Google.Protobuf.WellKnownTypes.Empty());
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

//        public static void Terminate()
//        {
//            RemoveServiceDiscoveryListener(_serviceDiscoveryListener);
//            TerminateInternal();
//            MetricsReporters.Terminate();
//        }
//
        public static Server GetServerById(string serverId)
        {
            try{
                var protoSv = _client.GetServer(new NPitaya.Protos.Server{Id=serverId});
                return protoSv;
            } catch (Exception _){
                return null;
            }
        }
//
        // TODO deprecate this frontendId field, it's not useful when using nats
        public static unsafe Task<PushResponse> SendPushToUser(string serverType, string route, string uid,
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
                
                return _client.SendPush(new PushRequest{FrontendType=serverType, Push=push});
            });
        }

        public static unsafe Task<PushResponse> SendKickToUser(string serverType, KickMsg kick)
        {
            return _rpcTaskFactory.StartNew(() =>
            {
                return _client.SendKick(new KickRequest{FrontendType=serverType, Kick=kick});
            });
        }

        public static unsafe Task<T> Rpc<T>(string serverId, Route route, object msg)
        {
            return _rpcTaskFactory.StartNew(() =>
            {
                var retError = new Error();
                var ok = false;
                Response res = null;
                Stopwatch sw = null;
                try
                {
                    var data = SerializerUtils.SerializeOrRaw(msg, _serializer);
                    sw = Stopwatch.StartNew();
                    fixed (byte* p = data)
                    {
                        // TODO this can be optimized I think by using a readonly span
                        res = _client.SendRPC(new RequestTo{ServerID=serverId, Msg=new Msg{Route=route.ToString(), Data=ByteString.CopyFrom(data.AsSpan()), Type=MsgType.MsgRequest}});
                    }

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
