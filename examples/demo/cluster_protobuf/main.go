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
	"github.com/topfreegames/pitaya/serialize/protobuf"
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
	ws := acceptor.NewWSAcceptor(fmt.Sprintf(":%d", port), "/pitaya")
	pitaya.Register(&services.Connector{},
		component.WithName("connector"),
		component.WithNameFunc(strings.ToLower),
	)
	pitaya.RegisterRemote(&services.ConnectorRemote{},
		component.WithName("connectorremote"),
		component.WithNameFunc(strings.ToLower),
	)

	pitaya.AddRoute("room", func(
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

	pitaya.AddAcceptor(ws)
}

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := flag.String("type", "connector", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")

	flag.Parse()

	defer (func() {
		pitaya.Shutdown()
	})()

	pitaya.SetSerializer(protobuf.NewSerializer())
	pitaya.SetServerType(*svType)

	if !*isFrontend {
		configureBackend()
	} else {
		configureFrontend(*port)
	}

	pitaya.Configure(*isFrontend, *svType, pitaya.Cluster)
	pitaya.Start()
}
