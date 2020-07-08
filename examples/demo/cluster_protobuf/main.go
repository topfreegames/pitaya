package main

import (
	"flag"
	"fmt"

	"strings"

	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/examples/demo/cluster_protobuf/services"
	"github.com/topfreegames/pitaya/serialize/protobuf"
)

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
	ws := acceptor.NewWSAcceptor(fmt.Sprintf(":%d", port))
	pitaya.Register(&services.Connector{},
		component.WithName("connector"),
		component.WithNameFunc(strings.ToLower),
	)
	pitaya.RegisterRemote(&services.ConnectorRemote{},
		component.WithName("connectorremote"),
		component.WithNameFunc(strings.ToLower),
	)

	app.AddAcceptor(ws)
}

var app pitaya.Pitaya

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := flag.String("type", "connector", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")

	flag.Parse()

	app := pitaya.NewApp(*isFrontend, *svType, pitaya.Cluster, map[string]string{})

	defer app.Shutdown()

	ser := protobuf.NewSerializer()
	app.SetSerializer(ser)

	if !*isFrontend {
		configureBackend()
	} else {
		configureFrontend(*port)
	}

	app.Start()
}
