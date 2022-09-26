using System;

namespace NPitaya.Serializer
{
    public interface ISerializer
    {
        byte[] Marshal(object o);
        object Unmarshal(byte[] bytes, Type t);
    }
}