using System;
using System.Text;
using PitayaSimpleJson;

namespace NPitaya.Serializer
{
    public class JSONSerializer: ISerializer
    {
        public byte[] Marshal(object o)
        {
            return Encoding.UTF8.GetBytes(PitayaSimpleJson.SimpleJson.SerializeObject(o));
        }

        public object Unmarshal(byte[] bytes, Type t)
        {
            if (bytes.Length == 0) bytes = Encoding.UTF8.GetBytes("{}");
            return PitayaSimpleJson.SimpleJson.DeserializeObject(Encoding.UTF8.GetString(bytes), t);
        }
    }
}