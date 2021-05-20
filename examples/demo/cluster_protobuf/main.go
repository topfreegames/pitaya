package main

import (
	"flag"
	"fmt"

	"strings"

	"github.com/topfreegames/pitaya/examples/demo/cluster_protobuf/services"
	pitaya "github.com/topfreegames/pitaya/pkg"
	"github.com/topfreegames/pitaya/pkg/acceptor"
	"github.com/topfreegames/pitaya/pkg/component"
	"github.com/topfreegames/pitaya/pkg/serialize/protobuf"
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
}

func configureFrontend(connectorType string, port int) {
	var connector acceptor.Acceptor
	if connectorType == "websocket" {
		connector = acceptor.NewWSAcceptor(fmt.Sprintf(":%d", port))
	} else if connectorType == "tcp" {
		connector = acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", port))
	}
	pitaya.Register(&services.Connector{},
		component.WithName("connector"),
		component.WithNameFunc(strings.ToLower),
	)
	pitaya.RegisterRemote(&services.ConnectorRemote{},
		component.WithName("connectorremote"),
		component.WithNameFunc(strings.ToLower),
	)

	pitaya.AddAcceptor(connector)
}

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := flag.String("type", "connector", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")
	connector := flag.String("connector", "tcp", "the type of connector, valid values: [\"websocket\",\"tcp\"]")

	flag.Parse()

	defer pitaya.Shutdown()

	ser := protobuf.NewSerializer()

	pitaya.SetSerializer(ser)

	if !*isFrontend {
		configureBackend()
	} else {
		configureFrontend(*connector, *port)
	}

	pitaya.Configure(*isFrontend, *svType, pitaya.Cluster, map[string]string{})
	pitaya.Start()
}
