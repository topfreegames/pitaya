package main

import (
	"flag"
	"fmt"

	"strings"

	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/examples/demo/cluster/services"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/serialize/json"
	"github.com/topfreegames/pitaya/session"
)

func configureBackend() {
	room := services.NewRoom()
	pitaya.Register(room,
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	pitaya.RegisterRemote(room,
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	// traffic stats
	pitaya.AfterHandler(room.Stats.Outbound)
	pitaya.BeforeHandler(room.Stats.Inbound)
}

func configureFrontend(port int) {
	ws := acceptor.NewWSAcceptor(fmt.Sprintf(":%d", port))
	tcp := acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", port+1))

	pitaya.Register(&services.Connector{},
		component.WithName("connector"),
		component.WithNameFunc(strings.ToLower),
	)
	pitaya.RegisterRemote(&services.ConnectorRemote{},
		component.WithName("connectorremote"),
		component.WithNameFunc(strings.ToLower),
	)

	err := pitaya.AddRoute("room", func(
		session *session.Session,
		route *route.Route,
		servers map[string]*cluster.Server,
	) (*cluster.Server, error) {
		// will return the first server
		for k := range servers {
			return servers[k], nil
		}
		return nil, nil
	})

	if err != nil {
		fmt.Printf("error adding route %s\n", err.Error())
	}

	err = pitaya.SetDictionary(map[string]uint16{
		"connector.getsessiondata": 1,
		"connector.setsessiondata": 2,
		"room.room.getsessiondata": 3,
		"onMessage":                4,
		"onMembers":                5,
	})

	if err != nil {
		fmt.Printf("error setting route dictionary %s\n", err.Error())
	}

	pitaya.AddAcceptor(ws)
	pitaya.AddAcceptor(tcp)
}

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := flag.String("type", "connector", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")

	flag.Parse()

	defer pitaya.Shutdown()

	pitaya.SetSerializer(json.NewSerializer())

	if !*isFrontend {
		configureBackend()
	} else {
		configureFrontend(*port)
	}

	pitaya.Configure(*isFrontend, *svType, pitaya.Cluster, map[string]string{})
	pitaya.Start()
}
