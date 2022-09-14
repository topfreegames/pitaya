package pkg

import (
	"github.com/google/uuid"
	"github.com/topfreegames/pitaya/v2/pkg/acceptor"
	"github.com/topfreegames/pitaya/v2/pkg/agent"
	cluster2 "github.com/topfreegames/pitaya/v2/pkg/cluster"
	config2 "github.com/topfreegames/pitaya/v2/pkg/config"
	codec2 "github.com/topfreegames/pitaya/v2/pkg/conn/codec"
	"github.com/topfreegames/pitaya/v2/pkg/conn/message"
	"github.com/topfreegames/pitaya/v2/pkg/defaultpipelines"
	groups2 "github.com/topfreegames/pitaya/v2/pkg/groups"
	"github.com/topfreegames/pitaya/v2/pkg/logger"
	"github.com/topfreegames/pitaya/v2/pkg/metrics"
	"github.com/topfreegames/pitaya/v2/pkg/metrics/models"
	"github.com/topfreegames/pitaya/v2/pkg/pipeline"
	"github.com/topfreegames/pitaya/v2/pkg/router"
	"github.com/topfreegames/pitaya/v2/pkg/serialize"
	"github.com/topfreegames/pitaya/v2/pkg/serialize/json"
	"github.com/topfreegames/pitaya/v2/pkg/service"
	"github.com/topfreegames/pitaya/v2/pkg/session"
	"github.com/topfreegames/pitaya/v2/pkg/worker"
)

// Builder holds dependency instances for a pitaya App
type Builder struct {
	acceptors        []acceptor.Acceptor
	Config           config2.BuilderConfig
	DieChan          chan bool
	PacketDecoder    codec2.PacketDecoder
	PacketEncoder    codec2.PacketEncoder
	MessageEncoder   *message.MessagesEncoder
	Serializer       serialize.Serializer
	Router           *router.Router
	RPCClient        cluster2.RPCClient
	RPCServer        cluster2.RPCServer
	MetricsReporters []metrics.Reporter
	Server           *cluster2.Server
	ServerMode       ServerMode
	ServiceDiscovery cluster2.ServiceDiscovery
	Groups           groups2.GroupService
	SessionPool      session.SessionPool
	Worker           *worker.Worker
	HandlerHooks     *pipeline.HandlerHooks
}

// PitayaBuilder Builder interface
type PitayaBuilder interface {
	Build() Pitaya
}

// NewBuilderWithConfigs return a builder instance with default dependency instances for a pitaya App
// with configs defined by a config file (config.Config) and default paths (see documentation).
func NewBuilderWithConfigs(
	isFrontend bool,
	serverType string,
	serverMode ServerMode,
	serverMetadata map[string]string,
	conf *config2.Config,
) *Builder {
	builderConfig := config2.NewBuilderConfig(conf)
	customMetrics := config2.NewCustomMetricsSpec(conf)
	prometheusConfig := config2.NewPrometheusConfig(conf)
	statsdConfig := config2.NewStatsdConfig(conf)
	etcdSDConfig := config2.NewEtcdServiceDiscoveryConfig(conf)
	natsRPCServerConfig := config2.NewNatsRPCServerConfig(conf)
	natsRPCClientConfig := config2.NewNatsRPCClientConfig(conf)
	workerConfig := config2.NewWorkerConfig(conf)
	enqueueOpts := config2.NewEnqueueOpts(conf)
	groupServiceConfig := config2.NewMemoryGroupConfig(conf)
	return NewBuilder(
		isFrontend,
		serverType,
		serverMode,
		serverMetadata,
		*builderConfig,
		*customMetrics,
		*prometheusConfig,
		*statsdConfig,
		*etcdSDConfig,
		*natsRPCServerConfig,
		*natsRPCClientConfig,
		*workerConfig,
		*enqueueOpts,
		*groupServiceConfig,
	)
}

// NewDefaultBuilder return a builder instance with default dependency instances for a pitaya App,
// with default configs
func NewDefaultBuilder(isFrontend bool, serverType string, serverMode ServerMode, serverMetadata map[string]string, builderConfig config2.BuilderConfig) *Builder {
	customMetrics := config2.NewDefaultCustomMetricsSpec()
	prometheusConfig := config2.NewDefaultPrometheusConfig()
	statsdConfig := config2.NewDefaultStatsdConfig()
	etcdSDConfig := config2.NewDefaultEtcdServiceDiscoveryConfig()
	natsRPCServerConfig := config2.NewDefaultNatsRPCServerConfig()
	natsRPCClientConfig := config2.NewDefaultNatsRPCClientConfig()
	workerConfig := config2.NewDefaultWorkerConfig()
	enqueueOpts := config2.NewDefaultEnqueueOpts()
	groupServiceConfig := config2.NewDefaultMemoryGroupConfig()
	return NewBuilder(
		isFrontend,
		serverType,
		serverMode,
		serverMetadata,
		builderConfig,
		*customMetrics,
		*prometheusConfig,
		*statsdConfig,
		*etcdSDConfig,
		*natsRPCServerConfig,
		*natsRPCClientConfig,
		*workerConfig,
		*enqueueOpts,
		*groupServiceConfig,
	)
}

// NewBuilder return a builder instance with default dependency instances for a pitaya App,
// with configs explicitly defined
func NewBuilder(isFrontend bool,
	serverType string,
	serverMode ServerMode,
	serverMetadata map[string]string,
	config config2.BuilderConfig,
	customMetrics models.CustomMetricsSpec,
	prometheusConfig config2.PrometheusConfig,
	statsdConfig config2.StatsdConfig,
	etcdSDConfig config2.EtcdServiceDiscoveryConfig,
	natsRPCServerConfig config2.NatsRPCServerConfig,
	natsRPCClientConfig config2.NatsRPCClientConfig,
	workerConfig config2.WorkerConfig,
	enqueueOpts config2.EnqueueOpts,
	groupServiceConfig config2.MemoryGroupConfig,
) *Builder {
	server := cluster2.NewServer(uuid.New().String(), serverType, isFrontend, serverMetadata)
	dieChan := make(chan bool)

	metricsReporters := []metrics.Reporter{}
	if config.Metrics.Prometheus.Enabled {
		metricsReporters = addDefaultPrometheus(prometheusConfig, customMetrics, metricsReporters, serverType)
	}

	if config.Metrics.Statsd.Enabled {
		metricsReporters = addDefaultStatsd(statsdConfig, metricsReporters, serverType)
	}

	handlerHooks := pipeline.NewHandlerHooks()
	if config.DefaultPipelines.StructValidation.Enabled {
		configureDefaultPipelines(handlerHooks)
	}

	sessionPool := session.NewSessionPool()

	var serviceDiscovery cluster2.ServiceDiscovery
	var rpcServer cluster2.RPCServer
	var rpcClient cluster2.RPCClient
	if serverMode == Cluster {
		var err error
		serviceDiscovery, err = cluster2.NewEtcdServiceDiscovery(etcdSDConfig, server, dieChan)
		if err != nil {
			logger.Log.Fatalf("error creating default cluster service discovery component: %s", err.Error())
		}

		rpcServer, err = cluster2.NewNatsRPCServer(natsRPCServerConfig, server, metricsReporters, dieChan, sessionPool)
		if err != nil {
			logger.Log.Fatalf("error setting default cluster rpc server component: %s", err.Error())
		}

		rpcClient, err = cluster2.NewNatsRPCClient(natsRPCClientConfig, server, metricsReporters, dieChan)
		if err != nil {
			logger.Log.Fatalf("error setting default cluster rpc client component: %s", err.Error())
		}
	}

	worker, err := worker.NewWorker(workerConfig, enqueueOpts)
	if err != nil {
		logger.Log.Fatalf("error creating default worker: %s", err.Error())
	}

	gsi := groups2.NewMemoryGroupService(groupServiceConfig)
	if err != nil {
		panic(err)
	}

	return &Builder{
		acceptors:        []acceptor.Acceptor{},
		Config:           config,
		DieChan:          dieChan,
		PacketDecoder:    codec2.NewPomeloPacketDecoder(),
		PacketEncoder:    codec2.NewPomeloPacketEncoder(),
		MessageEncoder:   message.NewMessagesEncoder(config.Pitaya.Handler.Messages.Compression),
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

	agentFactory := agent.NewAgentFactory(builder.DieChan,
		builder.PacketDecoder,
		builder.PacketEncoder,
		builder.Serializer,
		builder.Config.Pitaya.Heartbeat.Interval,
		builder.MessageEncoder,
		builder.Config.Pitaya.Buffer.Agent.Messages,
		builder.SessionPool,
		builder.MetricsReporters,
	)

	handlerService := service.NewHandlerService(
		builder.PacketDecoder,
		builder.Serializer,
		builder.Config.Pitaya.Buffer.Handler.LocalProcess,
		builder.Config.Pitaya.Buffer.Handler.RemoteProcess,
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
		builder.Config.Pitaya,
	)
}

// NewDefaultApp returns a default pitaya app instance
func NewDefaultApp(isFrontend bool, serverType string, serverMode ServerMode, serverMetadata map[string]string, config config2.BuilderConfig) Pitaya {
	builder := NewDefaultBuilder(isFrontend, serverType, serverMode, serverMetadata, config)
	return builder.Build()
}

func configureDefaultPipelines(handlerHooks *pipeline.HandlerHooks) {
	handlerHooks.BeforeHandler.PushBack(defaultpipelines.StructValidatorInstance.Validate)
}

func addDefaultPrometheus(config config2.PrometheusConfig, customMetrics models.CustomMetricsSpec, reporters []metrics.Reporter, serverType string) []metrics.Reporter {
	prometheus, err := CreatePrometheusReporter(serverType, config, customMetrics)
	if err != nil {
		logger.Log.Errorf("failed to start prometheus metrics reporter, skipping %v", err)
	} else {
		reporters = append(reporters, prometheus)
	}
	return reporters
}

func addDefaultStatsd(config config2.StatsdConfig, reporters []metrics.Reporter, serverType string) []metrics.Reporter {
	statsd, err := CreateStatsdReporter(serverType, config)
	if err != nil {
		logger.Log.Errorf("failed to start statsd metrics reporter, skipping %v", err)
	} else {
		reporters = append(reporters, statsd)
	}
	return reporters
}
