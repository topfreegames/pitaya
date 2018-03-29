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
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/component"
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

// TODO this NEEDS better configuration
const (
	processRemoteMsgConcurrency = 10
	dispatchConcurrency         = 10
)

// App is the base app struct
type App struct {
	server           *cluster.Server
	debug            bool
	startAt          time.Time
	dieChan          chan bool
	acceptors        []acceptor.Acceptor
	heartbeat        time.Duration
	packetDecoder    codec.PacketDecoder
	packetEncoder    codec.PacketEncoder
	serializer       serialize.Serializer
	serviceDiscovery cluster.ServiceDiscovery
	rpcServer        cluster.RPCServer
	rpcClient        cluster.RPCClient
	router           *router.Router
	serverMode       ServerMode
	onSessionBind    func(*session.Session)
	configured       bool
}

var (
	app = &App{
		server: &cluster.Server{
			ID:       uuid.New().String(),
			Type:     "game",
			Data:     map[string]string{},
			Frontend: true,
		},
		debug:         false,
		startAt:       time.Now(),
		dieChan:       make(chan bool),
		acceptors:     []acceptor.Acceptor{},
		heartbeat:     30 * time.Second,
		packetDecoder: codec.NewPomeloPacketDecoder(),
		packetEncoder: codec.NewPomeloPacketEncoder(),
		serverMode:    Standalone,
		serializer:    json.NewSerializer(),
		configured:    false,
		router:        router.New(),
	}
	log = logger.Log

	remoteService  *service.RemoteService
	handlerService *service.HandlerService
)

// Configure configures the app
func Configure(isFrontend bool, serverType string, serverMode ServerMode) {
	if app.configured {
		log.Warn("pitaya configured twice!")
	}
	app.server.Frontend = isFrontend
	app.server.Type = serverType
	app.serverMode = serverMode
	app.configured = true
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
	//TODO
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

// SetServerType sets the server type
// TODO need to specify in start
func SetServerType(t string) {
	app.server.Type = t
}

// SetServerData sets the server data that will be broadcasted using service discovery to other servers
// TODO need to specify in start
func SetServerData(data map[string]string) {
	app.server.Data = data
}

func startDefaultSD() {
	// initialize default service discovery
	// TODO remove this, force specifying
	var err error
	app.serviceDiscovery, err = cluster.NewEtcdServiceDiscovery(
		[]string{"localhost:2379"},
		time.Duration(5)*time.Second,
		"pitaya/",
		time.Duration(20)*time.Second,
		time.Duration(60)*time.Second,
		time.Duration(120)*time.Second,
		app.server,
	)
	if err != nil {
		log.Fatalf("error starting cluster service discovery component: %s", err.Error())
	}
}

func startDefaultRPCServer() {
	// initialize default rpc server
	// TODO remove this, force specifying
	var err error
	SetRPCServer(cluster.NewNatsRPCServer(
		"nats://localhost:4222",
		app.server,
	))
	if err != nil {
		log.Fatalf("error starting cluster rpc server component: %s", err.Error())
	}
}

func startDefaultRPCClient() {
	// initialize default rpc client
	// TODO remove this, force specifying
	var err error
	app.rpcClient = cluster.NewNatsRPCClient(
		"nats://localhost:4222",
		app.server,
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
			log.Warn("creating default service discovery because cluster mode is enabled, if you want to specify yours, use pitaya.SetServiceDiscoveryClient")
			startDefaultSD()
		}
		if app.rpcServer == nil {
			log.Warn("creating default rpc server because cluster mode is enabled, if you want to specify yours, use pitaya.SetRPCServer")
			startDefaultRPCServer()
		}
		if app.rpcClient == nil {
			log.Warn("creating default rpc client because cluster mode is enabled, if you want to specify yours, use pitaya.SetRPCClient")
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
		app.server,
		remoteService,
		dispatchConcurrency,
	)

	listen()

	defer timer.GlobalTicker.Stop()

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
	for i := 0; i < dispatchConcurrency; i++ {
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
		// TODO should this be done this way?
		for i := 0; i < processRemoteMsgConcurrency; i++ {
			go remoteService.ProcessRemoteMessages(i)
		}
		// TODO: use same parellelism?
		go remoteService.ProcessUserPush()
	}
}

// SetDictionary set routes map, TODO(warning): set dictionary in runtime would be a dangerous operation!!!!!!
func SetDictionary(dict map[string]uint16) {
	message.SetDictionary(dict)
}

// AddRoute adds a routing function to a server type
// TODO calling this method with the server already running is VERY dangerous
func AddRoute(
	serverType string,
	routingFunction func(
		session *session.Session,
		route *route.Route,
		servers map[string]*cluster.Server,
	) (*cluster.Server, error),
) {
	if app.router != nil {
		app.router.AddRoute(serverType, routingFunction)
	} else {
		log.Warn("ignoring route add as app router is nil")
	}
}

// Shutdown send a signal to let 'pitaya' shutdown itself.
func Shutdown() {
	close(app.dieChan)
}
