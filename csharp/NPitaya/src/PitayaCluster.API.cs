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

        public enum ServiceDiscoveryAction
        {
            ServerAdded,
            ServerRemoved,
        }

        public class ServiceDiscoveryListener
        {
            public Action<ServiceDiscoveryAction, Server> onServer;
            public IntPtr NativeListenerHandle { get; set; }
            public ServiceDiscoveryListener(Action<ServiceDiscoveryAction, Server> onServer)
            {
                Debug.Assert(onServer != null);
                this.onServer = onServer;
                NativeListenerHandle = IntPtr.Zero;
            }
        }

        private static ServiceDiscoveryListener _serviceDiscoveryListener;
        private static GCHandle _serviceDiscoveryListenerHandle;

        // Sidecar stuff
        // TODO create configuration on buffer sizes
        
        static BlockingCollection<SidecarRequest> queueRead = new BlockingCollection<SidecarRequest>(1000);
        static BlockingCollection<RPCResponse> queueWrite = new BlockingCollection<RPCResponse>(1000);

        // TODO this should now be a pure csharp implementation of getting sigint/sigterm
        //public static void AddSignalHandler(Action cb)
        //{
        //    _onSignalEvent += cb;
        //    OnSignalInternal(OnSignal);
        //}

        //private static void OnSignal()
        //{
        //    Logger.Info("Invoking signal handler");
        //    _onSignalEvent?.Invoke();
        //}

        private static List<Type> GetAllInheriting(Type type)
        {
            return AppDomain.CurrentDomain.GetAssemblies().SelectMany(x => x.GetTypes())
                .Where(x => type.IsAssignableFrom(x) && !x.IsInterface && !x.IsAbstract && x.FullName != type.FullName)
                .Select(x => x).ToList();
        }

        private static void ListenToIncomingRPCs(Sidecar.SidecarClient client)
        {
            // TODO extract this as a method
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
                bool debug
                )
        {
            _client = InitializeSidecarClient(sidecarListenAddr, server, debug);

            if (_client == null)
            {
                throw new PitayaException("Initialization failed");
            }

            RegisterRemotesAndHandlers();

            ListenToIncomingRPCs(_client);
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
//        public static Server? GetServerById(string serverId)
//        {
//            var retServer = new Server();
//
//            bool ok = GetServerByIdInternal(serverId, ref retServer);
//
//            if (!ok)
//            {
//                Logger.Error($"There are no servers with id {serverId}");
//                return null;
//            }
//
//            //var server = (Pitaya.Server)Marshal.PtrToStructure(serverPtr, typeof(Pitaya.Server));
//            //FreeServerInternal(serverPtr);
//            return retServer;
//        }
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

//        private static void OnServerAddedOrRemovedNativeCb(int serverAdded, IntPtr serverPtr, IntPtr user)
//        {
//            var pitayaClusterHandle = (GCHandle)user;
//            var serviceDiscoveryListener = pitayaClusterHandle.Target as ServiceDiscoveryListener;
//
//            if (serviceDiscoveryListener == null)
//            {
//                Logger.Warn("The service discovery listener is null!");
//                return;
//            }
//
//            var server = (Server)Marshal.PtrToStructure(serverPtr, typeof(Server));
//
//            if (serverAdded == 1)
//                serviceDiscoveryListener.onServer(ServiceDiscoveryAction.ServerAdded, server);
//            else
//                serviceDiscoveryListener.onServer(ServiceDiscoveryAction.ServerRemoved, server);
//        }
//
//        private static void AddServiceDiscoveryListener(ServiceDiscoveryListener listener)
//        {
//            _serviceDiscoveryListener = listener;
//            if (listener == null)
//                return;
//
//            _serviceDiscoveryListenerHandle = GCHandle.Alloc(_serviceDiscoveryListener);
//
//            IntPtr nativeListenerHandle = tfg_pitc_AddServiceDiscoveryListener(
//                OnServerAddedOrRemovedNativeCb,
//                (IntPtr)_serviceDiscoveryListenerHandle
//            );
//
//            listener.NativeListenerHandle = nativeListenerHandle;
//        }
//
//        private static void RemoveServiceDiscoveryListener(ServiceDiscoveryListener listener)
//        {
//            if (listener != null)
//            {
//                tfg_pitc_RemoveServiceDiscoveryListener(listener.NativeListenerHandle);
//                _serviceDiscoveryListenerHandle.Free();
//                _serviceDiscoveryListener = null;
//            }
//        }
    }
}
