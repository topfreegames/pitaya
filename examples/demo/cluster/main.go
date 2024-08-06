package main

import (
	"context"
	"flag"
	"fmt"

	"strings"

	"github.com/sirupsen/logrus"
	"github.com/topfreegames/pitaya/v3/examples/demo/cluster/services"
	pitaya "github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/acceptor"
	"github.com/topfreegames/pitaya/v3/pkg/cluster"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/pkg/config"
	"github.com/topfreegames/pitaya/v3/pkg/groups"
	"github.com/topfreegames/pitaya/v3/pkg/route"
	"github.com/topfreegames/pitaya/v3/pkg/serialize/json"
	"github.com/topfreegames/pitaya/v3/pkg/serialize/protobuf"
	"github.com/topfreegames/pitaya/v3/pkg/tracing"
)

var app pitaya.Pitaya

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

	app.RegisterRemote(services.NewConnectorRemote(app),
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

func configureOpenTelemetry(logger logrus.FieldLogger) {
	err := tracing.InitializeOtel()
	if err != nil {
		logger.Errorf("Failed to initialize OpenTelemetry: %v", err)
	}
}

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := flag.String("type", "connector", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")
	serializer := flag.String("serializer", "json", "the serializer to use")

	flag.Parse()

	configureOpenTelemetry(logrus.New())

	builder := pitaya.NewDefaultBuilder(*isFrontend, *svType, pitaya.Cluster, map[string]string{}, *config.NewDefaultPitayaConfig())
	if *isFrontend {
		tcp := acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", *port))
		builder.AddAcceptor(tcp)
	}

	if *serializer == "json" {
		builder.Serializer = json.NewSerializer()
	} else if *serializer == "protobuf" {
		builder.Serializer = protobuf.NewSerializer()
	} else {
		panic("unknown serializer " + *serializer)
	}

	builder.Groups = groups.NewMemoryGroupService(builder.Config.Groups.Memory)
	app = builder.Build()

	defer app.Shutdown()

	if !*isFrontend {
		configureBackend()
	} else {
		configureFrontend(*port)
	}

	app.Start()
}
