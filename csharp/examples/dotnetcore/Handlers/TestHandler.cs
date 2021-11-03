using System;
using System.Runtime.Serialization;
using System.Text;
using System.Threading.Tasks;
using NPitaya.Models;
using NPitaya.Protos;
using NPitaya;

#pragma warning disable 1998
namespace exampleapp.Handlers
{
    class TestHandler : BaseHandler
    {
        public async Task<Protos.RPCRes> Entry(PitayaSession pitayaSession, Protos.RPCMsg msg)
        {
            Logger.Debug($"Received this msg {msg.Msg}");
            var response = new Protos.RPCRes
            {
                Msg = $"hello from csharp handler!!! :) {Guid.NewGuid().ToString()}",
                Code = 200
            };
            return response;
        }

        public async Task<NPitaya.Protos.Server> GetServer(PitayaSession session, Protos.RPCMsg msg)
        {
            return PitayaCluster.GetServerById(msg.Msg);
        }

        public async Task NotifyBind(PitayaSession pitayaSession, Protos.RPCMsg msg)
        {
            var response = new Protos.RPCRes
            {
                Msg = $"hello from csharp handler!!! :) {Guid.NewGuid().ToString()}",
                Code = 200
            };

            await pitayaSession.Bind("uidbla");

            Console.WriteLine("handler executed with arg {0}", msg);
            Console.WriteLine("handler executed with session ipversion {0}", pitayaSession.GetString("ipversion"));
        }
        public async Task SetSessionDataTest(PitayaSession pitayaSession, Protos.RPCMsg msg)
        {
            pitayaSession.Set("msg", "testingMsg");
            pitayaSession.Set("int", 3);
            pitayaSession.Set("double", 3.33);
            await pitayaSession.PushToFrontend();
        }

        public async Task TestPush(PitayaSession pitayaSession)
        {
            Console.WriteLine("got empty notify");
            var msg = Encoding.UTF8.GetBytes("test felipe");
            var res = await pitayaSession.Push(new Protos.RPCRes{Code = 200, Msg = "this is a test push"}, "connector", "test.route");
            if (res.HasFailed)
            {
                Logger.Error("push to user failed! {0}", res.FailedUids);
            }
        }
        
        public async Task TestKick(PitayaSession pitayaSession)
        {
            var res = await pitayaSession.Kick("connector");
            if (res.HasFailed)
            {
                Logger.Error("kick user failed! {0}", res.FailedUids);
            }
        }

        [DataContract]
        public class TestClass
        {
            [DataMember(Name = "msg")]
            public string Msg;
            [DataMember(Name = "code")]
            public int Code;
        }

        public async Task<TestClass> OnlyValidWithJson(PitayaSession s, TestClass t)
        {
            return t;
        }
    }
}
