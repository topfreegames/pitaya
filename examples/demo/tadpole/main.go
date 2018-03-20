package main

import (
	"log"
	"net/http"
	"os"

	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/examples/demo/tadpole/logic"
	"github.com/topfreegames/pitaya/serialize/json"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Name = "tadpole"
	app.Author = "nano Authors"
	app.Version = "0.0.1"
	app.Copyright = "nano Authors reserved"
	app.Usage = "tadpole"

	// flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "addr",
			Value: ":23456",
			Usage: "game server address",
		},
	}

	app.Action = serve

	app.Run(os.Args)
}

func serve(ctx *cli.Context) error {
	// register all service
	pitaya.Register(logic.NewManager())
	pitaya.Register(logic.NewWorld())
	pitaya.SetSerializer(json.NewSerializer())

	//pitaya.EnableDebug()
	log.SetFlags(log.LstdFlags | log.Llongfile)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	addr := ctx.String("addr")
	pitaya.AddAcceptor(acceptor.NewWSAcceptor(addr))
	pitaya.Configure(true, "tadpole", pitaya.Standalone)
	pitaya.Start()

	return nil
}
