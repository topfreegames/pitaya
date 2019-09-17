// Copyright (c) nano Author and TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package pitaya

import (
	"context"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"

	"time"

	"github.com/google/uuid"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/spf13/viper"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/conn/codec"
	"github.com/topfreegames/pitaya/conn/message"
	"github.com/topfreegames/pitaya/constants"
	pcontext "github.com/topfreegames/pitaya/context"
	"github.com/topfreegames/pitaya/defaultpipelines"
	"github.com/topfreegames/pitaya/docgenerator"
	"github.com/topfreegames/pitaya/errors"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/metrics"
	mods "github.com/topfreegames/pitaya/modules"
	"github.com/topfreegames/pitaya/remote"
	"github.com/topfreegames/pitaya/router"
	"github.com/topfreegames/pitaya/serialize"
	"github.com/topfreegames/pitaya/serialize/json"
	"github.com/topfreegames/pitaya/service"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/timer"
	"github.com/topfreegames/pitaya/tracing"
	"github.com/topfreegames/pitaya/worker"
)

// ServerMode represents a server mode
type ServerMode byte

const (
	_ ServerMode = iota
	// Cluster represents a server running with connection to other servers
	Cluster
	// Standalone represents a server running without connection to other servers
	Standalone
)

// App is the base app struct
type App struct {
	acceptors        []acceptor.Acceptor
	config           *config.Config
	configured       bool
	debug            bool
	dieChan          chan bool
	heartbeat        time.Duration
	onSessionBind    func(*session.Session)
	messageEncoder   message.Encoder
	packetDecoder    codec.PacketDecoder
	packetEncoder    codec.PacketEncoder
	router           *router.Router
	rpcClient        cluster.RPCClient
	rpcServer        cluster.RPCServer
	metricsReporters []metrics.Reporter
	running          bool
	serializer       serialize.Serializer
	server           *cluster.Server
	serverMode       ServerMode
	serviceDiscovery cluster.ServiceDiscovery
	startAt          time.Time
	worker           *worker.Worker
}

var (
	app = &App{
		server:           cluster.NewServer(uuid.New().String(), "game", true, map[string]string{}),
		debug:            false,
		startAt:          time.Now(),
		dieChan:          make(chan bool),
		acceptors:        []acceptor.Acceptor{},
		packetDecoder:    codec.NewPomeloPacketDecoder(),
		packetEncoder:    codec.NewPomeloPacketEncoder(),
		metricsReporters: make([]metrics.Reporter, 0),
		serverMode:       Standalone,
		serializer:       json.NewSerializer(),
		configured:       false,
		running:          false,
		router:           router.New(),
	}

	remoteService  *service.RemoteService
	handlerService *service.HandlerService
)

// Configure configures the app
func Configure(
	isFrontend bool,
	serverType string,
	serverMode ServerMode,
	serverMetadata map[string]string,
	cfgs ...*viper.Viper,
) {
	if app.configured {
		logger.Log.Warn("pitaya configured twice!")
	}
	app.config = config.NewConfig(cfgs...)
	if app.heartbeat == time.Duration(0) {
		app.heartbeat = app.config.GetDuration("pitaya.heartbeat.interval")
	}
	app.server.Frontend = isFrontend
	app.server.Type = serverType
	app.serverMode = serverMode
	app.server.Metadata = serverMetadata
	app.messageEncoder = message.NewMessagesEncoder(app.config.GetBool("pitaya.handler.messages.compression"))
	configureMetrics(serverType)
	configureDefaultPipelines(app.config)
	app.configured = true
}

func configureMetrics(serverType string) {
	app.metricsReporters = make([]metrics.Reporter, 0)
	constTags := app.config.GetStringMapString("pitaya.metrics.constTags")

	if app.config.GetBool("pitaya.metrics.prometheus.enabled") {
		port := app.config.GetInt("pitaya.metrics.prometheus.port")
		logger.Log.Infof("prometheus is enabled, configuring reporter on port %d", port)
		prometheus, err := metrics.GetPrometheusReporter(serverType, app.config, constTags)
		if err != nil {
			logger.Log.Errorf("failed to start prometheus metrics reporter, skipping %v", err)
		} else {
			AddMetricsReporter(prometheus)
		}
	} else {
		logger.Log.Info("prometheus is disabled, reporter will not be enabled")
	}

	if app.config.GetBool("pitaya.metrics.statsd.enabled") {
		logger.Log.Infof(
			"statsd is enabled, configuring the metrics reporter with host: %s",
			app.config.Get("pitaya.metrics.statsd.host"),
		)
		metricsReporter, err := metrics.NewStatsdReporter(
			app.config,
			serverType,
			constTags,
		)
		if err != nil {
			logger.Log.Errorf("failed to start statds metrics reporter, skipping %v", err)
		} else {
			logger.Log.Info("successfully configured statsd metrics reporter")
			AddMetricsReporter(metricsReporter)
		}
	}
}

func configureDefaultPipelines(config *config.Config) {
	if config.GetBool("pitaya.defaultpipelines.structvalidation.enabled") {
		BeforeHandler(defaultpipelines.StructValidatorInstance.Validate)
	}
}

// AddAcceptor adds a new acceptor to app
func AddAcceptor(ac acceptor.Acceptor) {
	if !app.server.Frontend {
		logger.Log.Error("tried to add an acceptor to a backend server, skipping")
		return
	}
	app.acceptors = append(app.acceptors, ac)
}

// GetDieChan gets the channel that the app sinalizes when its going to die
func GetDieChan() chan bool {
	return app.dieChan
}

// SetDebug toggles debug on/off
func SetDebug(debug bool) {
	app.debug = debug
}

// SetPacketDecoder changes the decoder used to parse messages received
func SetPacketDecoder(d codec.PacketDecoder) {
	app.packetDecoder = d
}

// SetPacketEncoder changes the encoder used to package outgoing messages
func SetPacketEncoder(e codec.PacketEncoder) {
	app.packetEncoder = e
}

// SetHeartbeatTime sets the heartbeat time
func SetHeartbeatTime(interval time.Duration) {
	app.heartbeat = interval
}

// SetLogger logger setter
func SetLogger(l logger.Logger) {
	logger.Log = l
}

// GetServerID returns the generated server id
func GetServerID() string {
	return app.server.ID
}

// GetConfig gets the pitaya config instance
func GetConfig() *config.Config {
	return app.config
}

// GetMetricsReporters gets registered metrics reporters
func GetMetricsReporters() []metrics.Reporter {
	return app.metricsReporters
}

// SetRPCServer to be used
func SetRPCServer(s cluster.RPCServer) {
	app.rpcServer = s
}

// SetRPCClient to be used
func SetRPCClient(s cluster.RPCClient) {
	app.rpcClient = s
}

// SetServiceDiscoveryClient to be used
func SetServiceDiscoveryClient(s cluster.ServiceDiscovery) {
	app.serviceDiscovery = s
}

// SetSerializer customize application serializer, which automatically Marshal
// and UnMarshal handler payload
func SetSerializer(seri serialize.Serializer) {
	app.serializer = seri
}

// GetSerializer gets the app serializer
func GetSerializer() serialize.Serializer {
	return app.serializer
}

// GetServer gets the local server instance
func GetServer() *cluster.Server {
	return app.server
}

// GetServerByID returns the server with the specified id
func GetServerByID(id string) (*cluster.Server, error) {
	return app.serviceDiscovery.GetServer(id)
}

// GetServersByType get all servers of type
func GetServersByType(t string) (map[string]*cluster.Server, error) {
	return app.serviceDiscovery.GetServersByType(t)
}

// GetServers get all servers
func GetServers() []*cluster.Server {
	return app.serviceDiscovery.GetServers()
}

// AddMetricsReporter to be used
func AddMetricsReporter(mr metrics.Reporter) {
	app.metricsReporters = append(app.metricsReporters, mr)
}

func startDefaultSD() {
	// initialize default service discovery
	var err error
	app.serviceDiscovery, err = cluster.NewEtcdServiceDiscovery(
		app.config,
		app.server,
		app.dieChan,
	)
	if err != nil {
		logger.Log.Fatalf("error starting cluster service discovery component: %s", err.Error())
	}
}

func startDefaultRPCServer() {
	// initialize default rpc server
	rpcServer, err := cluster.NewNatsRPCServer(app.config, app.server, app.metricsReporters, app.dieChan)
	if err != nil {
		logger.Log.Fatalf("error starting cluster rpc server component: %s", err.Error())
	}
	SetRPCServer(rpcServer)
}

func startDefaultRPCClient() {
	// initialize default rpc client
	rpcClient, err := cluster.NewNatsRPCClient(app.config, app.server, app.metricsReporters, app.dieChan)
	if err != nil {
		logger.Log.Fatalf("error starting cluster rpc client component: %s", err.Error())
	}
	SetRPCClient(rpcClient)
}

func initSysRemotes() {
	sys := &remote.Sys{}
	RegisterRemote(sys,
		component.WithName("sys"),
		component.WithNameFunc(strings.ToLower),
	)
}

func periodicMetrics() {
	period := app.config.GetDuration("pitaya.metrics.periodicMetrics.period")
	go metrics.ReportSysMetrics(app.metricsReporters, period)

	if app.worker.Started() {
		go worker.Report(app.metricsReporters, period)
	}
}

// Start starts the app
func Start() {
	if !app.configured {
		logger.Log.Fatal("starting app without configuring it first! call pitaya.Configure()")
	}

	if !app.server.Frontend && len(app.acceptors) > 0 {
		logger.Log.Fatal("acceptors are not allowed on backend servers")
	}

	if app.server.Frontend && len(app.acceptors) == 0 {
		logger.Log.Fatal("frontend servers should have at least one configured acceptor")
	}

	if app.serverMode == Cluster {
		if app.serviceDiscovery == nil {
			logger.Log.Warn("creating default service discovery because cluster mode is enabled, " +
				"if you want to specify yours, use pitaya.SetServiceDiscoveryClient")
			startDefaultSD()
		}
		if app.rpcServer == nil {
			logger.Log.Warn("creating default rpc server because cluster mode is enabled, " +
				"if you want to specify yours, use pitaya.SetRPCServer")
			startDefaultRPCServer()
		}
		if app.rpcClient == nil {
			logger.Log.Warn("creating default rpc client because cluster mode is enabled, " +
				"if you want to specify yours, use pitaya.SetRPCClient")
			startDefaultRPCClient()
		}

		if reflect.TypeOf(app.rpcClient) == reflect.TypeOf(&cluster.GRPCClient{}) {
			app.serviceDiscovery.AddListener(app.rpcClient.(*cluster.GRPCClient))
		}

		if err := RegisterModuleBefore(app.rpcServer, "rpcServer"); err != nil {
			logger.Log.Fatal("failed to register rpc server module: %s", err.Error())
		}
		if err := RegisterModuleBefore(app.rpcClient, "rpcClient"); err != nil {
			logger.Log.Fatal("failed to register rpc client module: %s", err.Error())
		}
		// set the service discovery as the last module to be started to ensure
		// all modules have been properly initialized before the server starts
		// receiving requests from other pitaya servers
		if err := RegisterModuleAfter(app.serviceDiscovery, "serviceDiscovery"); err != nil {
			logger.Log.Fatal("failed to register service discovery module: %s", err.Error())
		}

		app.router.SetServiceDiscovery(app.serviceDiscovery)

		remoteService = service.NewRemoteService(
			app.rpcClient,
			app.rpcServer,
			app.serviceDiscovery,
			app.packetEncoder,
			app.serializer,
			app.router,
			app.messageEncoder,
			app.server,
		)

		app.rpcServer.SetPitayaServer(remoteService)

		initSysRemotes()
	}

	handlerService = service.NewHandlerService(
		app.dieChan,
		app.packetDecoder,
		app.packetEncoder,
		app.serializer,
		app.heartbeat,
		app.config.GetInt("pitaya.buffer.agent.messages"),
		app.config.GetInt("pitaya.buffer.handler.localprocess"),
		app.config.GetInt("pitaya.buffer.handler.remoteprocess"),
		app.server,
		remoteService,
		app.messageEncoder,
		app.metricsReporters,
	)

	periodicMetrics()

	listen()

	defer func() {
		timer.GlobalTicker.Stop()
		app.running = false
	}()

	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)

	// stop server
	select {
	case <-app.dieChan:
		logger.Log.Warn("the app will shutdown in a few seconds")
	case s := <-sg:
		logger.Log.Warn("got signal: ", s, ", shutting down...")
		close(app.dieChan)
	}

	logger.Log.Warn("server is stopping...")

	session.CloseAll()
	shutdownModules()
	shutdownComponents()
}

func listen() {
	startupComponents()
	// create global ticker instance, timer precision could be customized
	// by SetTimerPrecision
	timer.GlobalTicker = time.NewTicker(timer.Precision)

	logger.Log.Infof("starting server %s:%s", app.server.Type, app.server.ID)
	for i := 0; i < app.config.GetInt("pitaya.concurrency.handler.dispatch"); i++ {
		go handlerService.Dispatch(i)
	}
	for _, acc := range app.acceptors {
		a := acc
		go func() {
			for conn := range a.GetConnChan() {
				go handlerService.Handle(conn)
			}
		}()

		go func() {
			a.ListenAndServe()
		}()

		logger.Log.Infof("listening with acceptor %s on addr %s", reflect.TypeOf(a), a.GetAddr())
	}

	if app.serverMode == Cluster && app.server.Frontend && app.config.GetBool("pitaya.session.unique") {
		unique := mods.NewUniqueSession(app.server, app.rpcServer, app.rpcClient)
		remoteService.AddRemoteBindingListener(unique)
		RegisterModule(unique, "uniqueSession")
	}

	startModules()

	logger.Log.Info("all modules started!")

	app.running = true
}

// SetDictionary sets routes map
func SetDictionary(dict map[string]uint16) error {
	if app.running {
		return constants.ErrChangeDictionaryWhileRunning
	}
	return message.SetDictionary(dict)
}

// AddRoute adds a routing function to a server type
func AddRoute(
	serverType string,
	routingFunction router.RoutingFunc,
) error {
	if app.router != nil {
		if app.running {
			return constants.ErrChangeRouteWhileRunning
		}
		app.router.AddRoute(serverType, routingFunction)
	} else {
		return constants.ErrRouterNotInitialized
	}
	return nil
}

// Shutdown send a signal to let 'pitaya' shutdown itself.
func Shutdown() {
	select {
	case <-app.dieChan: // prevent closing closed channel
	default:
		close(app.dieChan)
	}
}

// Error creates a new error with a code, message and metadata
func Error(err error, code string, metadata ...map[string]string) *errors.Error {
	return errors.NewError(err, code, metadata...)
}

// GetSessionFromCtx retrieves a session from a given context
func GetSessionFromCtx(ctx context.Context) *session.Session {
	sessionVal := ctx.Value(constants.SessionCtxKey)
	if sessionVal == nil {
		logger.Log.Debug("ctx doesn't contain a session, are you calling GetSessionFromCtx from inside a remote?")
		return nil
	}
	return sessionVal.(*session.Session)
}

// GetDefaultLoggerFromCtx returns the default logger from the given context
func GetDefaultLoggerFromCtx(ctx context.Context) logger.Logger {
	l := ctx.Value(constants.LoggerCtxKey)
	if l == nil {
		return logger.Log
	}

	return l.(logger.Logger)
}

// AddMetricTagsToPropagateCtx adds a key and metric tags that will
// be propagated through RPC calls. Use the same tags that are at
// 'pitaya.metrics.additionalTags' config
func AddMetricTagsToPropagateCtx(
	ctx context.Context,
	tags map[string]string,
) context.Context {
	return pcontext.AddToPropagateCtx(ctx, constants.MetricTagsKey, tags)
}

// AddToPropagateCtx adds a key and value that will be propagated through RPC calls
func AddToPropagateCtx(ctx context.Context, key string, val interface{}) context.Context {
	return pcontext.AddToPropagateCtx(ctx, key, val)
}

// GetFromPropagateCtx adds a key and value that came through RPC calls
func GetFromPropagateCtx(ctx context.Context, key string) interface{} {
	return pcontext.GetFromPropagateCtx(ctx, key)
}

// ExtractSpan retrieves an opentracing span context from the given context
// The span context can be received directly or via an RPC call
func ExtractSpan(ctx context.Context) (opentracing.SpanContext, error) {
	return tracing.ExtractSpan(ctx)
}

// Documentation returns handler and remotes documentacion
func Documentation(getPtrNames bool) (map[string]interface{}, error) {
	handlerDocs, err := handlerService.Docs(getPtrNames)
	if err != nil {
		return nil, err
	}
	remoteDocs, err := remoteService.Docs(getPtrNames)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"handlers": handlerDocs,
		"remotes":  remoteDocs,
	}, nil
}

// AddGRPCInfoToMetadata adds host, external host and
// port into metadata
func AddGRPCInfoToMetadata(
	metadata map[string]string,
	region string,
	host, port string,
	externalHost, externalPort string,
) map[string]string {
	metadata[constants.GRPCHostKey] = host
	metadata[constants.GRPCPortKey] = port
	metadata[constants.GRPCExternalHostKey] = externalHost
	metadata[constants.GRPCExternalPortKey] = externalPort
	metadata[constants.RegionKey] = region
	return metadata
}

// Descriptor returns the protobuf message descriptor for a given message name
func Descriptor(protoName string) ([]byte, error) {
	return docgenerator.ProtoDescriptors(protoName)
}

// StartWorker configures, starts and returns pitaya worker
func StartWorker(config *config.Config) error {
	var err error
	app.worker, err = worker.NewWorker(config)
	if err != nil {
		return err
	}

	app.worker.Start()

	return nil
}

// RegisterRPCJob registers rpc job to execute jobs with retries
func RegisterRPCJob(rpcJob worker.RPCJob) error {
	err := app.worker.RegisterRPCJob(rpcJob)
	return err
}
