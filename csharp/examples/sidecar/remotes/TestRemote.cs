using System.Threading.Tasks;
using NPitaya.Models;

namespace exampleapp.remotes
{
#pragma warning disable 1998
    class TestRemote : BaseRemote
    {
        public async Task<Protos.RPCMsg> Remote(Protos.RPCMsg msg)
        {
            var response = new Protos.RPCMsg
            {
                Msg = $"hello from csharp :) {System.Guid.NewGuid().ToString()}"
            };
            return response;
        }
    }
}
