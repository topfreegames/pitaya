using System.Text;
using NPitaya.Models;
using Xunit;

namespace NPitayaTest.Tests.Models
{
    public class PitayaConfigurationTests
    {
        [Fact]
        public void Pitaya_Config_Instance_Is_Not_Null()
        {
            Assert.NotNull(PitayaConfiguration.Config);
        }

        [Fact]
        public void Pitaya_Config_Contains_Default_Values()
        {
            var config = new PitayaConfiguration();
            foreach(var kv in config.defaultValues){
                Assert.Equal(config.defaultValues[kv.Key], config.get(kv.Key));
            }
        }

        [Fact]
        public void Pitaya_Config_Get_Set_Int()
        {
            var config = new PitayaConfiguration();
            config.putInt("Test", 1);
            Assert.Equal(config.getInt("Test"), 1);
            config.putInt("Test2", 0);
            Assert.Equal(config.getInt("Test2"), 0);
        }

        [Fact]
        public void Pitaya_Config_Get_Set_String()
        {
            var config = new PitayaConfiguration();
            config.putString("Test3", "bla");
            Assert.Equal(config.getString("Test3"), "bla");
            config.putString("Test4", "");
            Assert.Equal(config.getString("Test4"), "");
        }

        [Fact]
        public void Pitaya_Config_With_Env_Var()
        {
            var configKey = "test.var";
            System.Environment.SetEnvironmentVariable(configKey.ToUpper().Replace(".","_"), "3000");
            var config = new PitayaConfiguration(new System.Collections.Generic.Dictionary<string, object>{
                {configKey, 1}
            });
            Assert.Equal(config.getInt(configKey), 3000);
        }
    }
}