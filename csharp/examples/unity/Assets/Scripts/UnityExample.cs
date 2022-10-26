using System.Collections.Generic;
using UnityEngine;
using UnityEngine.UI;
using NPitaya;
using System;

public class UnityExample : MonoBehaviour
{
    public Button startBtn;
    public Button terminateBtn;
    // Start is called before the first frame update
    void Start()
    {
        startBtn.onClick.AddListener(OnStartClick);
        terminateBtn.onClick.AddListener(OnTerminateClick);
    }

    void MyLog(NPitaya.Models.LogLevel level, string message, params object[] objs){
      Debug.Log("Hello focking: " + message);
    }

    void OnApplicationQuit()
    {
        Debug.Log("Application ending after " + Time.time + " seconds");
        PitayaCluster.Terminate();
    }
    void OnTerminateClick(){
        PitayaCluster.Terminate();
    }
    void OnStartClick()
    {
        NPitaya.Models.Logger.SetLevel(NPitaya.Models.LogLevel.DEBUG);
        string serverId = System.Guid.NewGuid().ToString();
        var metadata = new Dictionary<string, string>(){
            {"testKey", "testValue"}
        };

        var sv = new NPitaya.Protos.Server
        {
            Id = serverId,
            Type = "csharp",
            Hostname = "localhost",
            Frontend = false
        };
        sv.Metadata.Add(metadata);

        Debug.Log("Will initialize pitaya");
        PitayaCluster.SetSerializer(new NPitaya.Serializer.JSONSerializer());
        PitayaCluster.Initialize(
            "localhost",
            5000,
            sv,
            true,
            (sdEvent) =>
            {
                switch (sdEvent.Event)
                {
                    case NPitaya.Protos.SDEvent.Types.Event.Add:
                        Debug.Log("Server was added");
                        Debug.Log("   id: " + sdEvent.Server.Id);
                        Debug.Log(" type: " + sdEvent.Server.Type);
                        break;
                    case NPitaya.Protos.SDEvent.Types.Event.Remove:
                        Debug.Log("Server was removed");
                        Debug.Log("   id: " + sdEvent.Server.Id);
                        Debug.Log(" type: " + sdEvent.Server.Type);
                        break;
                    default:
                        throw new ArgumentOutOfRangeException(nameof(sdEvent.Event), sdEvent.Event, null);
                }
            }
          );
    }
}
