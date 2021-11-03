using System;
using Google.Protobuf;

namespace NPitaya.Serializer
{
    public class ProtobufSerializer: ISerializer
    {
        public byte[] Marshal(object o)
        {
            if (o == null) return new byte[]{};
            if (!(o is IMessage))
            {
                throw new PitayaException("Cannot serialize a type that doesn't implement IMessage");
            }

            if (o == null) return new byte[] { };

            return ((IMessage) o).ToByteArray();
        }

        public object Unmarshal(byte[] bytes, Type t)
        {
            if (!typeof(IMessage).IsAssignableFrom(t))
            {
                throw new PitayaException("Cannot deserialize to a type that doesn't implement IMessage");
                
            }
            var res = Activator.CreateInstance(t);
            ((IMessage)res).MergeFrom(bytes);
            return res;
        }
    }
}
