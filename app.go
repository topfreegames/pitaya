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
	"encoding/gob"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"

	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/remote"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/router"
	"github.com/topfreegames/pitaya/serialize"
	"github.com/topfreegames/pitaya/serialize/json"
	"github.com/topfreegames/pitaya/service"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/timer"
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
	packetDecoder    codec.PacketDecoder
	packetEncoder    codec.PacketEncoder
	router           *router.Router
	rpcClient        cluster.RPCClient
	rpcServer        cluster.RPCServer
	running          bool
	serializer       serialize.Serializer
	server           *cluster.Server
	serverMode       ServerMode
	serviceDiscovery cluster.ServiceDiscovery
	startAt          time.Time
}

var (
	app = &App{
		server: &cluster.Server{
			ID:       uuid.New().String(),
			Type:     "game",
			Metadata: map[string]string{},
			Frontend: true,
		},
		debug:         false,
		startAt:       time.Now(),
		dieChan:       make(chan bool),
		acceptors:     []acceptor.Acceptor{},
		packetDecoder: codec.NewPomeloPacketDecoder(),
		packetEncoder: codec.NewPomeloPacketEncoder(),
		serverMode:    Standalone,
		serializer:    json.NewSerializer(),
		configured:    false,
		running:       false,
		router:        router.New(),
	}
	log = logger.Log

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
		log.Warn("pitaya configured twice!")
	}
	app.config = config.NewConfig(cfgs...)
	if app.heartbeat == time.Duration(0) {
		app.heartbeat = app.config.GetDuration("pitaya.heartbeat.interval")
	}
	app.server.Frontend = isFrontend
	app.server.Type = serverType
	app.serverMode = serverMode
	app.configured = true
	app.server.Metadata = serverMetadata
}

// AddAcceptor adds a new acceptor to app
func AddAcceptor(ac acceptor.Acceptor) {
	if !app.server.Frontend {
		log.Error("tried to add an acceptor to a backend server, skipping")
		return
	}
	app.acceptors = append(app.acceptors, ac)
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

// SetRPCServer to be used
func SetRPCServer(s cluster.RPCServer) {
	app.rpcServer = s
	if reflect.TypeOf(s) == reflect.TypeOf(&cluster.NatsRPCServer{}) {
		// When using nats rpc server the server must start listening to messages
		// destined to the userID that's binding
		session.SetOnSessionBind(func(s *session.Session) error {
			if app.server.Frontend {
				subs, err := app.rpcServer.(*cluster.NatsRPCServer).SubscribeToUserMessages(s.UID())
				if err != nil {
					return err
				}
				s.Subscription = subs
			}
			return nil
		})
	}
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

func startDefaultSD() {
	// initialize default service discovery
	var err error
	app.serviceDiscovery, err = cluster.NewEtcdServiceDiscovery(
		app.config,
		app.server,
	)
	if err != nil {
		log.Fatalf("error starting cluster service discovery component: %s", err.Error())
	}
}

func startDefaultRPCServer() {
	// initialize default rpc server
	var err error
	rpcServer, err := cluster.NewNatsRPCServer(
		app.config,
		app.server,
	)
	if err != nil {
		log.Fatalf("error starting cluster rpc server component: %s", err.Error())
	}
	SetRPCServer(rpcServer)
}

func startDefaultRPCClient() {
	// initialize default rpc client
	var err error
	app.rpcClient = cluster.NewNatsRPCClient(
		app.config.GetString("pitaya.cluster.rpc.client.nats.connect"),
		app.server,
		app.config.GetDuration("pitaya.cluster.rpc.client.nats.requesttimeout"),
	)
	if err != nil {
		log.Fatalf("error starting cluster rpc client component: %s", err.Error())
	}
}

func initSysRemotes() {
	gob.Register(&session.Data{})
	sys := &remote.Sys{}
	RegisterRemote(sys,
		component.WithName("sys"),
		component.WithNameFunc(strings.ToLower),
	)
}

// Start starts the app
func Start() {
	if !app.configured {
		log.Fatal("starting app without configuring it first! call pitaya.Configure()")
	}

	if !app.server.Frontend && len(app.acceptors) > 0 {
		log.Fatal("acceptors are not allowed on backend servers")
	}

	if app.serverMode == Cluster {
		if app.serviceDiscovery == nil {
			log.Warn("creating default service discovery because cluster mode is enabled, " +
				"if you want to specify yours, use pitaya.SetServiceDiscoveryClient")
			startDefaultSD()
		}
		if app.rpcServer == nil {
			log.Warn("creating default rpc server because cluster mode is enabled, " +
				"if you want to specify yours, use pitaya.SetRPCServer")
			startDefaultRPCServer()
		}
		if app.rpcClient == nil {
			log.Warn("creating default rpc client because cluster mode is enabled, " +
				"if you want to specify yours, use pitaya.SetRPCClient")
			startDefaultRPCClient()
			RegisterModule(app.serviceDiscovery, "serviceDiscovery")
			RegisterModule(app.rpcServer, "rpcServer")
			RegisterModule(app.rpcClient, "rpcClient")
		}

		app.router.SetServiceDiscovery(app.serviceDiscovery)

		remoteService = service.NewRemoteService(
			app.rpcClient,
			app.rpcServer,
			app.serviceDiscovery,
			app.packetEncoder,
			app.serializer,
			app.router,
		)
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
	)

	listen()

	defer func() {
		timer.GlobalTicker.Stop()
		app.running = false
	}()

	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)

	// stop server
	select {
	case <-app.dieChan:
		log.Warn("The app will shutdown in a few seconds")
	case s := <-sg:
		log.Warn("got signal", s)
	}

	log.Warn("server is stopping...")

	shutdownModules()
	shutdownComponents()
}

func listen() {
	startupComponents()
	// create global ticker instance, timer precision could be customized
	// by SetTimerPrecision
	timer.GlobalTicker = time.NewTicker(timer.Precision)

	log.Infof("starting server %s:%s", app.server.Type, app.server.ID)
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

		log.Infof("listening with acceptor %s on addr %s", reflect.TypeOf(a), a.GetAddr())
	}

	startModules()

	// this handles remote messages
	if app.rpcServer != nil {
		for i := 0; i < app.config.GetInt("pitaya.concurrency.remote.service"); i++ {
			go remoteService.ProcessRemoteMessages(i)
		}
		// this should be so fast that we shoudn't need concurrency
		go remoteService.ProcessUserPush()
	}

	app.running = true
}

// SetDictionary set routes map
func SetDictionary(dict map[string]uint16) error {
	if app.running {
		return constants.ErrChangeDictionaryWhileRunning
	}
	return message.SetDictionary(dict)
}

// AddRoute adds a routing function to a server type
func AddRoute(
	serverType string,
	routingFunction func(
		session *session.Session,
		route *route.Route,
		servers map[string]*cluster.Server,
	) (*cluster.Server, error),
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
	close(app.dieChan)
}
