namespace NPitaya.Serializer
{
    public class SerializerUtils
    {
        public static byte[] SerializeOrRaw(object obj, ISerializer serializer)
        {
            if (obj is byte[])
            {
                return (byte[])obj;
            }

            return serializer.Marshal(obj);
        }
    }
}