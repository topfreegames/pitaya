using System.Reflection;
using System.Collections.Generic;
using System.Threading.Tasks;

namespace NPitaya.Models
{
    public class BaseHandler : IRemote
    {
        public string GetName()
        {
            return GetType().Name;
        }

        public Dictionary<string, RemoteMethod> GetRemotesMap()
        {
            Dictionary<string, RemoteMethod> dict = new Dictionary<string, RemoteMethod>();
            MethodBase[] methods = this.GetType().GetMethods(BindingFlags.Instance | BindingFlags.Public);
            foreach (var methodBase in methods)
            {
                var m = (MethodInfo) methodBase;
                if (m.IsPublic)
                {
                    if (typeof(Task).IsAssignableFrom(m.ReturnType))
                    {
                        var returnType = m.ReturnType.GenericTypeArguments.Length > 0
                            ? m.ReturnType.GenericTypeArguments[0]
                            : typeof(void);
                        ParameterInfo[] parameters = m.GetParameters();
                        if (parameters.Length == 2) // TODO need to use context
                        {
                            if (typeof(PitayaSession) ==
                                parameters[0].ParameterType && // TODO support bytes in and out, support context
                                (typeof(object).IsAssignableFrom(parameters[1].ParameterType)))
                            {
                                dict[m.Name] = new RemoteMethod(this, m, returnType, parameters[1].ParameterType);
                            }
                        }

                        if (parameters.Length == 1 && typeof(PitayaSession) == parameters[0].ParameterType)
                        {
                            dict[m.Name] = new RemoteMethod(this, m, returnType, null);
                        }
                    }
                }
            }

            return dict;
        }

        private static bool isValidHandler()
        {
            return true; //TODO implement this
        }
    }
}
