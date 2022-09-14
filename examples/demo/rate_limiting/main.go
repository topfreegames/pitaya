package main

import (
	"flag"
	"fmt"
	acceptor2 "github.com/topfreegames/pitaya/v2/pkg/acceptor"
	acceptorwrapper2 "github.com/topfreegames/pitaya/v2/pkg/acceptorwrapper"
	"github.com/topfreegames/pitaya/v2/pkg/component"
	config2 "github.com/topfreegames/pitaya/v2/pkg/config"
	"strings"
	"time"

	"github.com/spf13/viper"
	pitaya "github.com/topfreegames/pitaya/v2/pkg"
	"github.com/topfreegames/pitaya/v2/examples/demo/rate_limiting/services"
	"github.com/topfreegames/pitaya/v2/pkg/metrics"
)

func createAcceptor(port int, reporters []metrics.Reporter) acceptor2.Acceptor {

	// 5 requests in 1 minute. Doesn't make sense, just to test
	// rate limiting
	vConfig := viper.New()
	vConfig.Set("pitaya.conn.ratelimiting.limit", 5)
	vConfig.Set("pitaya.conn.ratelimiting.interval", time.Minute)
	pConfig := config2.NewConfig(vConfig)

	rateLimitConfig := config2.NewRateLimitingConfig(pConfig)

	tcp := acceptor2.NewTCPAcceptor(fmt.Sprintf(":%d", port))
	return acceptorwrapper2.WithWrappers(
		tcp,
		acceptorwrapper2.NewRateLimitingWrapper(reporters, *rateLimitConfig))
}

var app pitaya.Pitaya

func main() {
	port := flag.Int("port", 3250, "the port to listen")
	svType := "room"

	flag.Parse()

	config := config2.NewDefaultBuilderConfig()
	builder := pitaya.NewDefaultBuilder(true, svType, pitaya.Cluster, map[string]string{}, *config)
	builder.AddAcceptor(createAcceptor(*port, builder.MetricsReporters))

	app = builder.Build()

	defer app.Shutdown()

	room := services.NewRoom()
	app.Register(room,
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	app.Start()
}
