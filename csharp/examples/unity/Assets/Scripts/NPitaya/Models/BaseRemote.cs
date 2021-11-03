using System.Reflection;
using Google.Protobuf;
using System.Collections.Generic;
using System.Threading.Tasks;

namespace NPitaya.Models
{
    public class BaseRemote : IRemote
    {
        public string GetName()
        {
            return GetType().Name;
        }

        public Dictionary<string, RemoteMethod> GetRemotesMap()
        {
            Dictionary<string, RemoteMethod> dict = new Dictionary<string, RemoteMethod>();
            MethodBase[] methods = GetType().GetMethods(BindingFlags.Instance |
                                                        BindingFlags.Public);
            foreach (MethodInfo m in methods)
            {
                if (m.IsPublic)
                {
                    if (typeof(Task).IsAssignableFrom(m.ReturnType))
                    {
                        var returnType = m.ReturnType.GenericTypeArguments.Length > 0
                            ? m.ReturnType.GenericTypeArguments[0]
                            : typeof(void);
                        ParameterInfo[] parameters = m.GetParameters();
                        if (parameters.Length == 1)
                        {
                            if (typeof(object).IsAssignableFrom(parameters[0].ParameterType))
                            {
                                dict[m.Name] = new RemoteMethod(this, m, returnType,
                                    parameters[0].ParameterType);
                            }
                        }
                    }
                }
            }

            return dict;
        }

        private static bool isValidRemote()
        {
            return true;
        }
    }
}
