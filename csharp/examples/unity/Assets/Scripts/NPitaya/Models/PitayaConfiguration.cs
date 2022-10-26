using System.Collections.Generic;


namespace NPitaya.Models
{
    public class PitayaConfiguration
    {
        public const string CONFIG_HEARTBEAT_TIMEOUT_MS = "pitaya.sidecar.heartbeattimeoutms";
        public const string CONFIG_READBUFFER_SIZE = "pitaya.sidecar.readbuffersize";
        public const string CONFIG_WRITEBUFFER_SIZE = "pitaya.sidecar.writebuffersize";

        public static Dictionary<string, object> DefaultValues = new Dictionary<string,object>{
            {CONFIG_HEARTBEAT_TIMEOUT_MS, 5000},
            {CONFIG_READBUFFER_SIZE, 1000},
            {CONFIG_WRITEBUFFER_SIZE, 1000},
        };

        public Dictionary<string, object> defaultValues;
        private readonly Dictionary<string, object> configMap = new Dictionary<string, object>();

        public static PitayaConfiguration Config;

        private void initializeDefaultConfigurations(){
            foreach (var kv in defaultValues){
                configMap[kv.Key] = kv.Value;
            }
        }

        private string configNameToEnvVar(string configName){
            return configName.ToUpper().Replace(".", "_");
        }

        private void replaceConfigWithEnv(){
            foreach (var kv in defaultValues)
            {
                var envVarValue = System.Environment.GetEnvironmentVariable(configNameToEnvVar(kv.Key));
                if (envVarValue != null){
                    if(int.TryParse(kv.Value.ToString(), out var intVal)){
                        if (int.TryParse(envVarValue, out var ret)){
                            configMap[kv.Key] = ret;
                        } else {
                            Logger.Error("Tried to set int configuration: {0} with non valid number: {1}", kv.Key, envVarValue);
                        }
                    } else {
                        configMap[kv.Key] = envVarValue;
                    }
                }
            }
        }

        static PitayaConfiguration() {
            if (Config == null){
                Config = new PitayaConfiguration();
            }
        }

        public PitayaConfiguration(Dictionary<string, object> defaultValues){
            this.defaultValues = defaultValues;
            initializeDefaultConfigurations();
            replaceConfigWithEnv();
        }

        public PitayaConfiguration(){
            defaultValues = DefaultValues;
            initializeDefaultConfigurations();
            replaceConfigWithEnv();
        }

        public string getString(string key){
            return configMap[key].ToString();
        }

        public object get(string key){
            return configMap[key];
        }

        public int getInt(string key){
            int.TryParse(configMap[key].ToString(), out var ret);
            return ret;
        }

        public void putInt(string key, int value){
            configMap[key] = value;
        }

        public void putString(string key, string value){
            configMap[key] = value;
        }

        public void reset(){
            configMap.Clear();
            initializeDefaultConfigurations();
            replaceConfigWithEnv();
        }

    }

}