using System.Runtime.Serialization;

namespace NPitayaTest.Tests
{
    [DataContract]
    public class TestClass1
    {
        [DataMember(Name = "arg1")] public string Arg1;
        [DataMember(Name = "arg2")] public int Arg2;
        [DataMember(Name = "arg3")] public TestClass2 Arg3;

        public override bool Equals(object other)
        {
            if (other == null || other.GetType() != typeof(TestClass1)) return false;
            var otherT = (TestClass1) other;
            var res = Arg1.Equals(otherT.Arg1) && Arg2 == otherT.Arg2;
            if (Arg3 == null)
            {
                return res && otherT.Arg3 == null;
            }
            return res && Arg3.Equals(otherT.Arg3);
        }
    }

    [DataContract]
    public class TestClass2
    {
        [DataMember(Name = "arg3")] public string Arg3;
        [DataMember(Name = "arg4")] public int Arg4;

        public override bool Equals(object other)
        {
            if (other == null || other.GetType() != typeof(TestClass2)) return false;
            var otherT = (TestClass2) other;
            return (Arg3.Equals(otherT.Arg3) && Arg4 == otherT.Arg4);
        }
    }
}