using System;
using Grpc.Core;
using Protos;
using NPitaya.Protos;
using System.Threading;
using System.Threading.Tasks;
using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using System.Collections.Concurrent;

namespace SidecarClient
{
  // TODO fix this whole project to use npitaya
    class Program
    {

        // TODO: improve the bound sizes and make them configurable
        static BlockingCollection<SidecarRequest> queueRead = new BlockingCollection<SidecarRequest>(1000);
        static BlockingCollection<RPCResponse> queueWrite = new BlockingCollection<RPCResponse>(1000);
        // ReadRPCs will receive a stream of requests form the sidecar
        // During my benchmarks I could hit 100k req/sec with a golang server sending small messages in this stream
        // TODO: benchmark with larger messages
        public static void processRead(int i, Grpc.Core.AsyncDuplexStreamingCall<RPCResponse, SidecarRequest> stream)
        {
            Console.WriteLine("Starting read thread " + i);
            var task = new Task(() =>
            {
                while (true)
                {
                    var req = queueRead.Take();
                    var msg = new RPCMsg { Msg = "Hello there, response from rpc from c#" };
                    var res = new RPCResponse { ReqId = req.ReqId, Res = new Response { Data = msg.ToByteString() } };
                    queueWrite.Add(res);
                }
            });
            task.Start();

        }
        public static async void processWrite(Grpc.Core.AsyncDuplexStreamingCall<RPCResponse, SidecarRequest> stream)
        {
            while (true)
            {
                var res = queueWrite.Take();
                await stream.RequestStream.WriteAsync(res);
            }
        }

        public static async void HandleRPCs(Grpc.Core.AsyncDuplexStreamingCall<RPCResponse, SidecarRequest> stream)
        {
            var msgCount = 0;
            // Benchmark start
            Task benchmarkTask = new Task(() =>
                {
                    while (true)
                    {
                        Thread.Sleep(1000);
                        Console.WriteLine("msg/sec: " + msgCount);
                        msgCount = 0;
                    }
                });
            benchmarkTask.Start();

            // Benchmark end
            // TODO: Make this number of threads configurable
            for (int i = 0; i < Environment.ProcessorCount; i++)
            {
                processRead(i, stream);
            }

            // Only one write task because GRPC WriteAsync is not threadSafe and we can only have one pending write
            Task writeTask = new Task(() =>
            {
                processWrite(stream);
            });

            writeTask.Start();

            while (await stream.ResponseStream.MoveNext())
            {
                var current = stream.ResponseStream.Current;
                msgCount++;
                queueRead.Add(current);
            }
        }

        public static void Main(string[] args)
        {
            GrpcEnvironment.SetCompletionQueueCount(Environment.ProcessorCount);
            GrpcEnvironment.SetThreadPoolSize(Environment.ProcessorCount);
            //Channel channel = new Channel("127.0.0.1:3000", ChannelCredentials.Insecure);
            Channel channel = new Channel("unix:///tmp/pitaya.sock", ChannelCredentials.Insecure);

            var client = new Sidecar.SidecarClient(channel);
            var req = new StartPitayaRequest { Config = new NPitaya.Protos.Server { Frontend = false, Type = "csharp" }, DebugLog = true, ShouldCompressMessages = false};
            //
            client.StartPitaya(req);
            //
            Console.WriteLine("Press any key to start recv rpc...");
            Console.ReadKey();
            //
            var stream = client.ListenRPC();
            Task.Run(async () => HandleRPCs(stream));
            //
            //Console.WriteLine("Press any key to send rpc...");
            ////
            //var msg = new RPCMsg{Msg="Hello there, I'm csharp"};
            //var res = client.SendRPC(new Msg{Route="connector.connectorremote.remotefunc", Data=msg.ToByteString()});
            //var resMsg = new RPCMsg{};
            //resMsg.MergeFrom(res.Data);
            //Console.WriteLine("received this rpc answer: " + resMsg.Msg);
            //
            Console.WriteLine("Press any key to send stop...");
            Console.ReadKey();
            //
            client.StopPitaya(new Empty { });
            //
            Console.WriteLine("Press any key to exit...");
            Console.ReadKey();
            channel.ShutdownAsync().Wait();
        }
    }
}
