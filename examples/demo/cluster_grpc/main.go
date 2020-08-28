package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/spf13/viper"

	"strings"

	"github.com/topfreegames/pitaya/v2"
	"github.com/topfreegames/pitaya/v2/acceptor"
	"github.com/topfreegames/pitaya/v2/cluster"
	"github.com/topfreegames/pitaya/v2/component"
	"github.com/topfreegames/pitaya/v2/config"
	"github.com/topfreegames/pitaya/v2/constants"
	"github.com/topfreegames/pitaya/v2/examples/demo/cluster_grpc/services"
	"github.com/topfreegames/pitaya/v2/groups"
	"github.com/topfreegames/pitaya/v2/modules"
	"github.com/topfreegames/pitaya/v2/route"
)

func configureBackend() {
	room := services.NewRoom(app)
	app.Register(room,
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	app.RegisterRemote(room,
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)
}

func configureFrontend(port int) {
	app.Register(services.NewConnector(app),
		component.WithName("connector"),
		component.WithNameFunc(strings.ToLower),
	)
	app.RegisterRemote(&services.ConnectorRemote{},
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
}

var app pitaya.Pitaya

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := flag.String("type", "connector", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")
	rpcServerPort := flag.String("rpcsvport", "3434", "the port that grpc server will listen")

	flag.Parse()

	confs := viper.New()
	confs.Set("pitaya.cluster.rpc.server.grpc.port", *rpcServerPort)

	meta := map[string]string{
		constants.GRPCHostKey: "127.0.0.1",
		constants.GRPCPortKey: *rpcServerPort,
	}

	app, bs := createApp(*port, *isFrontend, *svType, meta, confs)

	defer app.Shutdown()

	app.RegisterModule(bs, "bindingsStorage")
	if !*isFrontend {
		configureBackend()
	} else {
		configureFrontend(*port)
	}
	app.Start()
}

func createApp(port int, isFrontend bool, svType string, meta map[string]string, confs ...*viper.Viper) (pitaya.Pitaya, *modules.ETCDBindingStorage) {
	builder := pitaya.NewBuilder(isFrontend, svType, pitaya.Cluster, meta, confs...)

	config := config.NewConfig(builder.Configs...)
	gs, err := cluster.NewGRPCServer(config, builder.Server, builder.MetricsReporters)
	if err != nil {
		panic(err)
	}
	builder.RPCServer = gs
	builder.Groups = groups.NewMemoryGroupService(config)

	bs := modules.NewETCDBindingStorage(builder.Server, builder.SessionPool, config)

	gc, err := cluster.NewGRPCClient(
		config,
		builder.Server,
		builder.MetricsReporters,
		bs,
		cluster.NewConfigInfoRetriever(config),
	)
	if err != nil {
		panic(err)
	}
	builder.RPCClient = gc

	if isFrontend {
		tcp := acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", port))
		builder.AddAcceptor(tcp)
	}

	return builder.Build(), bs
}
