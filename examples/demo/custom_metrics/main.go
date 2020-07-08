package main

import (
	"flag"
	"fmt"

	"strings"

	"github.com/spf13/viper"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/examples/demo/custom_metrics/services"
	"github.com/topfreegames/pitaya/serialize/json"
)

func configureRoom(port int) {
	tcp := acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", port))
	app.AddAcceptor(tcp)

	pitaya.Register(services.NewRoom(app),
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)
}

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

	app = pitaya.NewApp(isFrontend, svType, pitaya.Cluster, map[string]string{}, config)

	defer app.Shutdown()

	app.SetSerializer(json.NewSerializer())

	configureRoom(*port)
	app.Start()
}
