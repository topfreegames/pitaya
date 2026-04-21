package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/topfreegames/pitaya/v3/game/internal/connector"
	"github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/acceptor"
	"github.com/topfreegames/pitaya/v3/pkg/cluster"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/pkg/config"
	"github.com/topfreegames/pitaya/v3/pkg/groups"
	"github.com/topfreegames/pitaya/v3/pkg/session"
)

var app pitaya.Pitaya

func main() {
	port := flag.Int("port", 3250, "connector server port")
	flag.Parse()

	builder := pitaya.NewDefaultBuilder(
		true,
		"connector",
		pitaya.Cluster,
		map[string]string{},
		*config.NewDefaultPitayaConfig(),
	)

	wsAcceptor := acceptor.NewWSAcceptor(fmt.Sprintf(":%d", *port))
	builder.AddAcceptor(wsAcceptor)

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

	connectorHandler := connector.NewConnector(app)
	app.Register(connectorHandler, component.WithName("connector"))

	go startMetricsServer()

	fmt.Printf("Connector server starting on port %d...\n", *port)
	app.Start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Shutting down connector server...")
	app.Shutdown()
	fmt.Println("Connector server stopped")
}

func startMetricsServer() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	http.ListenAndServe(":8080", nil)
}
