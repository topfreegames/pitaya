using System;
using Grpc.Core;
using NPitaya.Models;
using NPitaya.Protos;
using System.Threading.Tasks;

namespace NPitaya
{
    public partial class PitayaCluster
    {
        private static bool _isInitialized = false;
        private static Sidecar.SidecarClient _client = null;
        private static Grpc.Core.Channel _channel;

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
            _isInitialized = true;
        }
        static void InitializePitaya(NPitaya.Protos.Server server, bool debug = false){
            var req = new StartPitayaRequest { Config = server, DebugLog = debug };
            _client.StartPitaya(req);
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
