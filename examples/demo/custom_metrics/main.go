package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/long12310225/pitaya/v2"
	"github.com/long12310225/pitaya/v2/acceptor"
	"github.com/long12310225/pitaya/v2/component"
	"github.com/long12310225/pitaya/v2/config"
	"github.com/long12310225/pitaya/v2/examples/demo/custom_metrics/services"
	"github.com/spf13/viper"
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
