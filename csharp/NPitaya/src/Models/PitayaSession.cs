using System;
using System.Collections.Generic;
using System.Threading.Tasks;
using Google.Protobuf;
using NPitaya.Constants;
using Json = PitayaSimpleJson.SimpleJson;
using NPitaya.Protos;

namespace NPitaya.Models
{
    public class PitayaSession
    {
        private Int64 _id;
        private string _frontendId;
        private Dictionary<string, object> _data;
        private string _rawData;
        public string RawData => _rawData;
        public string Uid { get; private set; }

        public PitayaSession(Protos.Session sessionProto)
        {
            _id = sessionProto.Id;
            Uid = sessionProto.Uid;
            _rawData = sessionProto.Data.ToStringUtf8();
            if (!String.IsNullOrEmpty(_rawData))
                _data = Json.DeserializeObject<Dictionary<string, object>>(_rawData);
        }

        public PitayaSession(Protos.Session sessionProto, string frontendId):this(sessionProto)
        {
            _frontendId = frontendId;
        }

        public override string ToString()
        {
            return $"ID: {_id}, UID: {Uid}, Data: {_rawData}";
        }

        public void Set(string key, object value)
        {
            _data[key] = value;
            _rawData = Json.SerializeObject(_data);
        }

        public object GetObject(string key)
        {
            if (!_data.ContainsKey(key))
            {
                throw new Exception($"key not found in session, parameter: {key}");
            }

            return _data[key];
        }

        public string GetString(string key)
        {
            return GetObject(key) as string;
        }

        public int GetInt(string key)
        {
            var obj = GetObject(key);
            return obj is int ? (int) obj : 0;
        }

        public double GetDouble(string key)
        {
            var obj = GetObject(key);
            return obj is double ? (double) obj : 0;
        }

        public Task PushToFrontend()
        {
            if (String.IsNullOrEmpty(_frontendId))
            {
                return Task.FromException(new Exception("cannot push to frontend, frontendId is invalid!"));
            }
            return SendRequestToFront(Routes.SessionPushRoute, true);
        }

        public Task Bind(string uid)
        {
            if (Uid != "")
            {
                return Task.FromException(new Exception("session already bound!"));
            }
            Uid = uid;
            // TODO only if server type is backend
            // TODO bind callbacks
            if (!string.IsNullOrEmpty(_frontendId)){
                return BindInFrontend();
            }

            return Task.CompletedTask;
        }

        private Task BindInFrontend()
        {
            return SendRequestToFront(Routes.SessionBindRoute, false);
        }

        private Task SendRequestToFront(string route, bool includeData)
        {
            var sessionProto = new Protos.Session
            {
                Id = _id,
                Uid = Uid
            };
            if (includeData)
            {
                sessionProto.Data = ByteString.CopyFromUtf8(_rawData);
            }
            return PitayaCluster.Rpc<Response>(_frontendId, Route.FromString(route), sessionProto.ToByteArray());
        }

        public Task<PushResponse> Push(object pushMsg, string svType, string route)
        {
            return PitayaCluster.SendPushToUser(svType, route, Uid, pushMsg);
        }

        public Task<PushResponse> Kick(string svType)
        {
            return PitayaCluster.SendKickToUser(svType, new KickMsg
            {
                UserId = Uid
            });
        }
    }
}
