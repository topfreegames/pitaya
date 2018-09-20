package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/spf13/viper"

	"strings"

	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/examples/demo/cluster_grpc/services"
	"github.com/topfreegames/pitaya/modules"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/serialize/json"
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

	err := pitaya.AddRoute("room", func(
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

	pitaya.AddAcceptor(tcp)
}

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := flag.String("type", "connector", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")
	rpcServerPort := flag.String("rpcsvport", "3434", "the port that grpc server will listen")

	flag.Parse()

	defer pitaya.Shutdown()

	pitaya.SetSerializer(json.NewSerializer())

	if !*isFrontend {
		configureBackend()
	} else {
		configureFrontend(*port)
	}

	confs := viper.New()
	confs.Set("pitaya.cluster.rpc.server.grpc.port", *rpcServerPort)

	meta := map[string]string{
		constants.GRPCHostKey: "127.0.0.1",
		constants.GRPCPortKey: *rpcServerPort,
	}

	pitaya.Configure(*isFrontend, *svType, pitaya.Cluster, meta, confs)
	gs, err := cluster.NewGRPCServer(pitaya.GetConfig(), pitaya.GetServer(), pitaya.GetMetricsReporters())
	if err != nil {
		panic(err)
	}

	bs := modules.NewETCDBindingStorage(pitaya.GetServer(), pitaya.GetConfig())
	pitaya.RegisterModule(bs, "bindingsStorage")

	gc, err := cluster.NewGRPCClient(
		pitaya.GetConfig(),
		pitaya.GetServer(),
		pitaya.GetMetricsReporters(),
		bs,
		cluster.NewConfigInfoRetriever(pitaya.GetConfig()),
	)
	if err != nil {
		panic(err)
	}
	pitaya.SetRPCServer(gs)
	pitaya.SetRPCClient(gc)
	pitaya.Start()
}
