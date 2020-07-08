package main

import (
	"context"
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
)

var app pitaya.Pitaya

func configureBackend() {
	room := services.NewRoom(app)
	pitaya.Register(room,
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	pitaya.RegisterRemote(room,
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)
}

func configureFrontend(port int) {
	tcp := acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", port))

	pitaya.Register(&services.Connector{},
		component.WithName("connector"),
		component.WithNameFunc(strings.ToLower),
	)
	pitaya.RegisterRemote(&services.ConnectorRemote{},
		component.WithName("connectorremote"),
		component.WithNameFunc(strings.ToLower),
	)

	err := app.AddRoute("room", func(
		ctx context.Context,
		route *route.Route,
		payload []byte,
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

	err = app.SetDictionary(map[string]uint16{
		"connector.getsessiondata": 1,
		"connector.setsessiondata": 2,
		"room.room.getsessiondata": 3,
		"onMessage":                4,
		"onMembers":                5,
	})

	if err != nil {
		fmt.Printf("error setting route dictionary %s\n", err.Error())
	}

	app.AddAcceptor(tcp)
}

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := flag.String("type", "connector", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")

	flag.Parse()

	app := pitaya.NewApp(*isFrontend, *svType, pitaya.Cluster, map[string]string{})

	defer app.Shutdown()

	app.SetSerializer(json.NewSerializer())

	if !*isFrontend {
		configureBackend()
	} else {
		configureFrontend(*port)
	}

	app.Start()
}
