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
	"github.com/topfreegames/pitaya/examples/demo/worker/services"
)

var app pitaya.Pitaya

func configureWorker() {
	worker := services.Worker{}
	worker.Configure(app)
}

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := flag.String("type", "metagame", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")

	flag.Parse()

	conf := viper.New()
	conf.SetDefault("pitaya.worker.redis.url", "localhost:6379")
	conf.SetDefault("pitaya.worker.redis.pool", "3")

	config := config.NewConfig(conf)

	tcp := acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", *port))

	builder := pitaya.NewBuilderWithConfigs(*isFrontend, *svType, pitaya.Cluster, map[string]string{}, config)
	if *isFrontend {
		builder.AddAcceptor(tcp)
	}
	app = builder.Build()

	defer app.Shutdown()

	defer app.Shutdown()

	switch *svType {
	case "metagame":
		app.RegisterRemote(&services.Metagame{},
			component.WithName("metagame"),
			component.WithNameFunc(strings.ToLower),
		)
	case "room":
		app.Register(services.NewRoom(app),
			component.WithName("room"),
			component.WithNameFunc(strings.ToLower),
		)
	case "worker":
		configureWorker()
	}

	app.Start()
}
