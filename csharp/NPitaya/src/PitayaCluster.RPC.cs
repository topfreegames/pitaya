using System;
using System.Diagnostics;
using System.Runtime.InteropServices;
using System.Threading.Tasks;
using Google.Protobuf;
using NPitaya.Metrics;
using NPitaya.Models;
using NPitaya.Serializer;
using NPitaya.Protos;
using static NPitaya.Utils.Utils;

namespace NPitaya
{
    public partial class PitayaCluster
    {
        private static async Task<Response> HandleIncomingRpc(Request req)
        {
            Response res = null;
            String route = req.Msg.Route;
            bool success = false;
            var sw = Stopwatch.StartNew();
            try{
                res = await RPCCbFuncImpl(req, sw);
                success = true;
                return res;
            }
            catch (Exception e)
            {
                var innerMostException = e;
                while (innerMostException.InnerException != null)
                    innerMostException = innerMostException.InnerException;

                Logger.Error("Exception thrown in handler, error:{0}",
                    innerMostException.Message); // TODO externalize method and only print stacktrace when debug
#if NPITAYA_DEBUG
                // If we are compiling with a Debug define, we want to print a stacktrace whenever a route
                // throws an exception.
                Logger.Error("Stacktrace: {0}", innerMostException.StackTrace);
#endif
                res = GetErrorResponse("PIT-500", innerMostException.Message);
                return res;
            }
            finally
            {
                if (success)
                {
                    MetricsReporters.ReportTimer(Metrics.Constants.Status.success.ToString(), route,
                        "handler", "", sw);
                }
                else
                {
                    MetricsReporters.ReportTimer(Metrics.Constants.Status.fail.ToString(), route,
                        "handler", "PIT-500", sw);
                }
            }
        }

        //private static Request BuildRequestData(IntPtr reqBufferPtr)
        //{
        //    var reqBuffer = (MemoryBuffer) Marshal.PtrToStructure(reqBufferPtr, typeof(MemoryBuffer));

        //    Request req = new Request();
        //    req.MergeFrom(new CodedInputStream(reqBuffer.GetData()));
        //    return req;
        //}

        private static async Task<Response> RPCCbFuncImpl(Request req, Stopwatch sw)
        {
            Response response;
            switch (req.Type)
            {
                case RPCType.User:
                    response = await HandleRpc(req, RPCType.User, sw);
                    break;
                case RPCType.Sys:
                    response = await HandleRpc(req, RPCType.Sys, sw);
                    break;
                default:
                    throw new Exception($"invalid rpc type, argument:{req.Type}");
            }

            return response;
        }

        internal static async Task<Response> HandleRpc(Protos.Request req, RPCType type, Stopwatch sw)
        {
            byte[] data = req.Msg.Data.ToByteArray();
            Route route = Route.FromString(req.Msg.Route);

            string handlerName = $"{route.service}.{route.method}";

            PitayaSession s = null;
            var response = new Response();

            RemoteMethod handler;
            if (type == RPCType.Sys)
            {
                s = new Models.PitayaSession(req.Session, req.FrontendID);
                if (!HandlersDict.ContainsKey(handlerName))
                {
                    response = GetErrorResponse("PIT-404",
                        $"remote/handler not found! remote/handler name: {handlerName}");
                    return response;
                }

                handler = HandlersDict[handlerName];
                MetricsReporters.ReportMessageProccessDelay(req.Msg.Route,"local", sw);
            }
            else
            {
                if (!RemotesDict.ContainsKey(handlerName))
                {
                    response = GetErrorResponse("PIT-404",
                        $"remote/handler not found! remote/handler name: {handlerName}");
                    return response;
                }

                handler = RemotesDict[handlerName];
                MetricsReporters.ReportMessageProccessDelay(req.Msg.Route,"remote", sw);
            }

            Task ans;
            if (handler.ArgType != null)
            {
                var arg = _serializer.Unmarshal(data, handler.ArgType);
                if (type == RPCType.Sys)
                    ans = handler.Method.Invoke(handler.Obj, new[] {s, arg}) as Task;
                else
                    ans = handler.Method.Invoke(handler.Obj, new[] {arg}) as Task;
            }
            else
            {
                if (type == RPCType.Sys)
                    ans = handler.Method.Invoke(handler.Obj, new object[] {s}) as Task;
                else
                    ans = handler.Method.Invoke(handler.Obj, new object[] { }) as Task;
            }

            await ans;
            byte[] ansBytes;

            if (handler.ReturnType != typeof(void))
            {
                ansBytes = SerializerUtils.SerializeOrRaw(ans.GetType().
                    GetProperty("Result")
                    ?.GetValue(ans), _serializer);
            }
            else
            {
                ansBytes = new byte[]{};
            }
            response.Data = ByteString.CopyFrom(ansBytes);
            return response;
        }
    }
}
