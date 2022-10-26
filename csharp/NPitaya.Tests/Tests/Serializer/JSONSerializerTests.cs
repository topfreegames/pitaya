using System.Text;
using NPitaya.Serializer;
using Xunit;

namespace NPitayaTest.Tests.Serializer
{
    public class JSONSerializerTests
    {
        [Fact]
        public void Marshal_Returns_Correct_Json()
        {
            var jsonSerializer = new JSONSerializer();
            var t1 = new TestClass1
            {
                Arg1 = "bla",
                Arg2 = 300
            };
            Assert.Equal("{\"arg1\":\"bla\",\"arg2\":300,\"arg3\":null}", Encoding.UTF8.GetString(jsonSerializer.Marshal(t1)));
        }
        
        [Fact]
        public void Marshal_Empty_Obj_Produces_Valid_Json()
        {
            var jsonSerializer = new JSONSerializer();
            var t1 = new TestClass1
            {
            };
            Assert.Equal("{\"arg1\":null,\"arg2\":0,\"arg3\":null}", Encoding.UTF8.GetString(jsonSerializer.Marshal(t1)));
        }
        
        [Fact]
        public void Unmarshal_Produces_Valid_Result()
        {
            var jsonSerializer = new JSONSerializer();
            var t1 = jsonSerializer.Unmarshal(Encoding.UTF8.GetBytes("{\"arg1\":\"bla\",\"arg2\":19}"), typeof(TestClass1));
            Assert.Equal(t1, (new TestClass1
            {
                Arg1 = "bla",
                Arg2 = 19
            }));
        }
        
        [Fact]
        public void Unmarshal_Composite_Struct_Produces_Valid_Result()
        {
            var jsonSerializer = new JSONSerializer();
            var t1 = jsonSerializer.Unmarshal(Encoding.UTF8.GetBytes("{\"arg1\":\"bla\",\"arg2\":19,\"arg3\":{\"arg3\":\"ola\",\"arg4\":400}}"), typeof(TestClass1));
            Assert.Equal(new TestClass1
            {
                Arg1 = "bla",
                Arg2 = 19,
                Arg3 = new TestClass2
                {
                    Arg3 = "ola",
                    Arg4 = 400
                }
            }, t1);
        }
    }
}