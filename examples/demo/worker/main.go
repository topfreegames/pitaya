package main

import (
	"flag"
	"fmt"

	"strings"

	"github.com/spf13/viper"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/examples/demo/worker/services"
	"github.com/topfreegames/pitaya/serialize/json"
)

func configureMetagame() {
	pitaya.RegisterRemote(&services.Metagame{},
		component.WithName("metagame"),
		component.WithNameFunc(strings.ToLower),
	)
}

func configureRoom(port int) error {
	tcp := acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", port))
	app.AddAcceptor(tcp)

	pitaya.Register(services.NewRoom(app),
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	err := app.StartWorker(app.GetConfig())
	return err
}

func configureWorker() error {
	worker := services.Worker{}
	err := worker.Configure(app)
	return err
}

var app pitaya.Pitaya

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := flag.String("type", "metagame", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")

	flag.Parse()

	config := viper.New()
	config.SetDefault("pitaya.worker.redis.url", "localhost:6379")
	config.SetDefault("pitaya.worker.redis.pool", "3")

	app := pitaya.NewApp(*isFrontend, *svType, pitaya.Cluster, map[string]string{})

	defer app.Shutdown()

	app.SetSerializer(json.NewSerializer())

	var err error
	switch *svType {
	case "metagame":
		configureMetagame()
	case "room":
		err = configureRoom(*port)
	case "worker":
		err = configureWorker()
	}

	if err != nil {
		panic(err)
	}

	app.Start()
}
