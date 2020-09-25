package main

import (
	"flag"
	"fmt"

	"strings"

	"github.com/spf13/viper"
	"github.com/tutumagi/pitaya"
	"github.com/tutumagi/pitaya/acceptor"
	"github.com/tutumagi/pitaya/component"
	"github.com/tutumagi/pitaya/examples/demo/worker/services"
	"github.com/tutumagi/pitaya/serialize/json"
)

func configureMetagame() {
	pitaya.RegisterRemote(&services.Metagame{},
		component.WithName("metagame"),
		component.WithNameFunc(strings.ToLower),
	)
}

func configureRoom(port int) error {
	tcp := acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", port))
	pitaya.AddAcceptor(tcp)

	pitaya.Register(&services.Room{},
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	err := pitaya.StartWorker(pitaya.GetConfig())
	return err
}

func configureWorker() error {
	worker := services.Worker{}
	err := worker.Configure()
	return err
}

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := flag.String("type", "metagame", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")

	flag.Parse()

	defer pitaya.Shutdown()

	pitaya.SetSerializer(json.NewSerializer())

	config := viper.New()
	config.SetDefault("pitaya.worker.redis.url", "localhost:6379")
	config.SetDefault("pitaya.worker.redis.pool", "3")

	pitaya.Configure(*isFrontend, *svType, pitaya.Cluster, map[string]string{})

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

	pitaya.Start()
}
