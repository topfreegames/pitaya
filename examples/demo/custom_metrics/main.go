package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"github.com/topfreegames/pitaya/v2"
	"github.com/topfreegames/pitaya/v2/acceptor"
	"github.com/topfreegames/pitaya/v2/component"
	"github.com/topfreegames/pitaya/v2/examples/demo/custom_metrics/services"
)

var app pitaya.Pitaya

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := "room"
	isFrontend := true

	flag.Parse()

	config := viper.New()
	config.AddConfigPath(".")
	config.SetConfigName("config")
	err := config.ReadInConfig()
	if err != nil {
		panic(err)
	}

	tcp := acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", *port))

	builder := pitaya.NewBuilder(isFrontend, svType, pitaya.Cluster, map[string]string{}, config)
	builder.AddAcceptor(tcp)
	app = builder.Build()

	defer app.Shutdown()

	app.Register(services.NewRoom(app),
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	app.Start()
}
