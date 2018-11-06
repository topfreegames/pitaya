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
	pitaya.AddAcceptor(tcp)

	pitaya.Register(&services.Room{},
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)
}

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := "room"
	isFrontend := true

	flag.Parse()

	defer pitaya.Shutdown()

	pitaya.SetSerializer(json.NewSerializer())

	config := viper.New()
	config.AddConfigPath(".")
	config.SetConfigName("config")
	err := config.ReadInConfig()
	if err != nil {
		panic(err)
	}

	pitaya.Configure(isFrontend, svType, pitaya.Cluster, map[string]string{}, config)
	configureRoom(*port)
	pitaya.Start()
}
