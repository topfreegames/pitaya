package pitaya

import (
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/agent"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/conn/codec"
	"github.com/topfreegames/pitaya/conn/message"
	"github.com/topfreegames/pitaya/defaultpipelines"
	"github.com/topfreegames/pitaya/groups"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/metrics"
	"github.com/topfreegames/pitaya/pipeline"
	"github.com/topfreegames/pitaya/router"
	"github.com/topfreegames/pitaya/serialize"
	"github.com/topfreegames/pitaya/serialize/json"
	"github.com/topfreegames/pitaya/service"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/worker"
)

// Builder holds dependency instances for a pitaya App
type Builder struct {
	acceptors        []acceptor.Acceptor
	Configs          []*viper.Viper
	DieChan          chan bool
	PacketDecoder    codec.PacketDecoder
	PacketEncoder    codec.PacketEncoder
	MessageEncoder   *message.MessagesEncoder
	Serializer       serialize.Serializer
	Router           *router.Router
	RPCClient        cluster.RPCClient
	RPCServer        cluster.RPCServer
	MetricsReporters []metrics.Reporter
	Server           *cluster.Server
	ServerMode       ServerMode
	ServiceDiscovery cluster.ServiceDiscovery
	Groups           groups.GroupService
	SessionPool      session.SessionPool
	Worker           *worker.Worker
	HandlerHooks     *pipeline.HandlerHooks
}

// PitayaBuilder Builder interface
type PitayaBuilder interface {
	Build() Pitaya
}

// NewBuilder return a builder instance with default dependency instances for a pitaya App
func NewBuilder(isFrontend bool, serverType string, serverMode ServerMode, serverMetadata map[string]string, cfgs ...*viper.Viper) *Builder {
	config := config.NewConfig(cfgs...)
	server := cluster.NewServer(uuid.New().String(), serverType, isFrontend, serverMetadata)
	dieChan := make(chan bool)

	metricsReporters := createDefaultMetrics(serverType, config)

	handlerHooks := pipeline.NewHandlerHooks()
	configureDefaultPipelines(handlerHooks, config)

	sessionPool := session.NewSessionPool()

	var serviceDiscovery cluster.ServiceDiscovery
	var rpcServer cluster.RPCServer
	var rpcClient cluster.RPCClient
	if serverMode == Cluster {
		var err error
		serviceDiscovery, err = cluster.NewEtcdServiceDiscovery(config, server, dieChan)
		if err != nil {
			logger.Log.Fatalf("error creating default cluster service discovery component: %s", err.Error())
		}

		rpcServer, err = cluster.NewNatsRPCServer(config, server, metricsReporters, dieChan, sessionPool)
		if err != nil {
			logger.Log.Fatalf("error setting default cluster rpc server component: %s", err.Error())
		}

		rpcClient, err = cluster.NewNatsRPCClient(config, server, metricsReporters, dieChan)
		if err != nil {
			logger.Log.Fatalf("error setting default cluster rpc client component: %s", err.Error())
		}
	}

	worker, err := worker.NewWorker(config)
	if err != nil {
		logger.Log.Fatalf("error creating default worker: %s", err.Error())
	}

	gsi := groups.NewMemoryGroupService(config)
	if err != nil {
		panic(err)
	}

	return &Builder{
		acceptors:        []acceptor.Acceptor{},
		Configs:          cfgs,
		DieChan:          dieChan,
		PacketDecoder:    codec.NewPomeloPacketDecoder(),
		PacketEncoder:    codec.NewPomeloPacketEncoder(),
		MessageEncoder:   message.NewMessagesEncoder(config.GetBool("pitaya.handler.messages.compression")),
		Serializer:       json.NewSerializer(),
		Router:           router.New(),
		RPCClient:        rpcClient,
		RPCServer:        rpcServer,
		MetricsReporters: metricsReporters,
		Server:           server,
		ServerMode:       serverMode,
		Groups:           gsi,
		HandlerHooks:     handlerHooks,
		ServiceDiscovery: serviceDiscovery,
		SessionPool:      sessionPool,
		Worker:           worker,
	}
}

// AddAcceptor adds a new acceptor to app
func (builder *Builder) AddAcceptor(ac acceptor.Acceptor) {
	if !builder.Server.Frontend {
		logger.Log.Error("tried to add an acceptor to a backend server, skipping")
		return
	}
	builder.acceptors = append(builder.acceptors, ac)
}

// Build returns a valid App instance
func (builder *Builder) Build() Pitaya {
	handlerPool := service.NewHandlerPool()
	var remoteService *service.RemoteService
	if builder.ServerMode == Standalone {
		if builder.ServiceDiscovery != nil || builder.RPCClient != nil || builder.RPCServer != nil {
			panic("Standalone mode can't have RPC or service discovery instances")
		}
	} else {
		if !(builder.ServiceDiscovery != nil && builder.RPCClient != nil && builder.RPCServer != nil) {
			panic("Cluster mode must have RPC and service discovery instances")
		}

		builder.Router.SetServiceDiscovery(builder.ServiceDiscovery)

		remoteService = service.NewRemoteService(
			builder.RPCClient,
			builder.RPCServer,
			builder.ServiceDiscovery,
			builder.PacketEncoder,
			builder.Serializer,
			builder.Router,
			builder.MessageEncoder,
			builder.Server,
			builder.SessionPool,
			builder.HandlerHooks,
			handlerPool,
		)

		builder.RPCServer.SetPitayaServer(remoteService)
	}

	config := config.NewConfig(builder.Configs...)

	agentFactory := agent.NewAgentFactory(builder.DieChan,
		builder.PacketDecoder,
		builder.PacketEncoder,
		builder.Serializer,
		config.GetDuration("pitaya.heartbeat.interval"),
		builder.MessageEncoder,
		config.GetInt("pitaya.buffer.agent.messages"),
		builder.SessionPool,
		builder.MetricsReporters,
	)

	handlerService := service.NewHandlerService(
		builder.PacketDecoder,
		builder.Serializer,
		config.GetInt("pitaya.buffer.handler.localprocess"),
		config.GetInt("pitaya.buffer.handler.remoteprocess"),
		builder.Server,
		remoteService,
		agentFactory,
		builder.MetricsReporters,
		builder.HandlerHooks,
		handlerPool,
	)

	return NewApp(
		builder.ServerMode,
		builder.Serializer,
		builder.acceptors,
		builder.DieChan,
		builder.Router,
		builder.Server,
		builder.RPCClient,
		builder.RPCServer,
		builder.Worker,
		builder.ServiceDiscovery,
		remoteService,
		handlerService,
		builder.Groups,
		builder.SessionPool,
		builder.MetricsReporters,
		builder.Configs...,
	)
}

// NewDefaultApp returns a default pitaya app instance
func NewDefaultApp(isFrontend bool, serverType string, serverMode ServerMode, serverMetadata map[string]string, cfgs ...*viper.Viper) Pitaya {
	builder := NewBuilder(isFrontend, serverType, serverMode, serverMetadata, cfgs...)
	return builder.Build()
}

func configureDefaultPipelines(handlerHooks *pipeline.HandlerHooks, config *config.Config) {
	if config.GetBool("pitaya.defaultpipelines.structvalidation.enabled") {
		handlerHooks.BeforeHandler.PushBack(defaultpipelines.StructValidatorInstance.Validate)
	}
}

func createDefaultMetrics(serverType string, config *config.Config) []metrics.Reporter {
	metricsReporters := make([]metrics.Reporter, 0)
	constTags := config.GetStringMapString("pitaya.metrics.constTags")

	if config.GetBool("pitaya.metrics.prometheus.enabled") {
		port := config.GetInt("pitaya.metrics.prometheus.port")
		logger.Log.Infof("prometheus is enabled, configuring reporter on port %d", port)
		prometheus, err := metrics.GetPrometheusReporter(serverType, config, constTags)
		if err != nil {
			logger.Log.Errorf("failed to start prometheus metrics reporter, skipping %v", err)
		} else {
			metricsReporters = append(metricsReporters, prometheus)
		}
	} else {
		logger.Log.Info("prometheus is disabled, reporter will not be enabled")
	}

	if config.GetBool("pitaya.metrics.statsd.enabled") {
		logger.Log.Infof(
			"statsd is enabled, configuring the metrics reporter with host: %s",
			config.Get("pitaya.metrics.statsd.host"),
		)
		metricsReporter, err := metrics.NewStatsdReporter(
			config,
			serverType,
			constTags,
		)
		if err != nil {
			logger.Log.Errorf("failed to start statds metrics reporter, skipping %v", err)
		} else {
			metricsReporters = append(metricsReporters, metricsReporter)
		}
	}
	return metricsReporters
}
