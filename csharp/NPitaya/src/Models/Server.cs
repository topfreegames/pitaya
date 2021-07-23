using System.Collections.Generic;

namespace NPitaya
{
    public struct Server
    {
        public string id;
        public string type;
        public Dictionary<string, string> metadata;
        public string hostname;
        public bool frontend;

        public Server(string id, string type, Dictionary<string, string> metadata, string hostname, bool frontend)
        {
            this.id = id;
            this.type = type;
            this.metadata = metadata;
            this.hostname = hostname;
            this.frontend = frontend;
        }
    }
}
