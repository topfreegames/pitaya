using System.Collections.Generic;
using NPitaya;

namespace NPitaya.Models
{
    public interface IRemote
    {
        string GetName();
        Dictionary<string, RemoteMethod> GetRemotesMap();
    }
}
