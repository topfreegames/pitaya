package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/examples/demo/custom_metrics/services"
)

var app pitaya.Pitaya

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := "room"
	isFrontend := true

	flag.Parse()

	cfg := viper.New()
	cfg.AddConfigPath(".")
	cfg.SetConfigName("config")
	err := cfg.ReadInConfig()
	if err != nil {
		panic(err)
	}

	tcp := acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", *port))

	conf := config.NewConfig(cfg)
	builder := pitaya.NewBuilderWithConfigs(isFrontend, svType, pitaya.Cluster, map[string]string{}, conf)
	builder.AddAcceptor(tcp)
	app = builder.Build()

	defer app.Shutdown()

	app.Register(services.NewRoom(app),
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	app.Start()
}
