package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/cluster"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/pkg/config"
	"github.com/topfreegames/pitaya/v3/pkg/groups"
	"github.com/topfreegames/pitaya/v3/pkg/route"
	"github.com/topfreegames/pitaya/v3/pkg/session"
)

var app pitaya.Pitaya

func main() {
	port := flag.Int("port", 3260, "room server port")
	flag.Parse()

	builder := pitaya.NewDefaultBuilder(
		false,
		"room",
		pitaya.Cluster,
		map[string]string{},
		*config.NewDefaultPitayaConfig(),
	)

	builder.Groups = groups.NewMemoryGroupService(builder.Config.Groups.Memory)

	natsClientConfig := config.NatsRPCClientConfig{
		Connect:        "nats://localhost:4222",
		RequestTimeout: 10,
	}
	natsServerConfig := config.NatsRPCServerConfig{
		Connect: "nats://localhost:4222",
		Buffer: struct {
			Messages int `mapstructure:"messages"`
			Push     int `mapstructure:"push"`
		}{
			Messages: 1024,
			Push:     1024,
		},
	}

	builder.SessionPool = session.NewSessionPool()

	app = builder.Build()
	defer app.Shutdown()

	natsClient, err := cluster.NewNatsRPCClient(natsClientConfig, app.GetServer(), nil, nil)
	if err != nil {
		fmt.Printf("Failed to create NATS RPC client: %v\n", err)
		os.Exit(1)
	}
	builder.RPCClient = natsClient

	natsServer, err := cluster.NewNatsRPCServer(natsServerConfig, app.GetServer(), nil, nil, builder.SessionPool)
	if err != nil {
		fmt.Printf("Failed to create NATS RPC server: %v\n", err)
		os.Exit(1)
	}
	builder.RPCServer = natsServer

	roomHandler := NewRoomHandler(app)
	app.Register(roomHandler, component.WithName("room"))

	err = app.AddRoute("room", func(
		ctx context.Context,
		route *route.Route,
		payload []byte,
		servers map[string]*cluster.Server,
	) (*cluster.Server, error) {
		for k := range servers {
			return servers[k], nil
		}
		return nil, nil
	})
	if err != nil {
		fmt.Printf("Error adding route: %v\n", err)
	}

	err = app.SetDictionary(map[string]uint16{
		"room.create":      1,
		"room.join":        2,
		"room.leave":       3,
		"room.get_info":    4,
		"room.list":        5,
		"room.move":        6,
		"room.attack":      7,
		"room.use_skill":   8,
	})
	if err != nil {
		fmt.Printf("Error setting dictionary: %v\n", err)
	}

	fmt.Printf("Room server starting on port %d...\n", *port)
	app.Start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Shutting down room server...")
	app.Shutdown()
	fmt.Println("Room server stopped")
}
