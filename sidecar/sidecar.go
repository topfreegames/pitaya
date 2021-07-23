package sidecar

import (
	"context"
	"net"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"sync/atomic"

	"github.com/sirupsen/logrus"
	pitaya "github.com/topfreegames/pitaya/pkg"
	"github.com/topfreegames/pitaya/pkg/cluster"
	"github.com/topfreegames/pitaya/pkg/config"
	"github.com/topfreegames/pitaya/pkg/constants"
	"github.com/topfreegames/pitaya/pkg/errors"
	"github.com/topfreegames/pitaya/pkg/logger"
	"github.com/topfreegames/pitaya/pkg/protos"
	"google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// TODO: implement jaeger into this, test everything, if connection dies this
// will go to hell, reconnection doesnt work, what can we do?  Bench results:
// 40k req/sec bidirectional stream, around 13k req/sec (RPC)

// TODO investigate why I will get drops in send rate every now and then during
// the benchmark test. I imagine maybe it's due to garbage collection?

// TODO I can utilize reutilizable objects, such as with a poll and reduce
// allocations here

// TODO document public methods and structs

// TODO fix the panic when the client disconnects

// Sidecar main struct to keep state
type Sidecar struct {
	config        *config.Config
	sidecarServer protos.SidecarServer
	stopChan      chan bool
	log           *logrus.Entry
	callChan      chan (*Call)
	shouldRun     bool
	listener      net.Listener
}

// Call struct represents an incoming RPC call from other servers
type Call struct {
	ctx   context.Context
	req   *protos.Request
	done  chan (bool)
	res   *protos.Response
	err   *protos.Error
	reqId uint64
}

// SidecarServer is the implementation of the GRPC server used to communicate
// with the sidecar client
type SidecarServer struct {
	protos.UnimplementedPitayaServer
}

var (
	sidecar = &Sidecar{
		stopChan:  make(chan bool),
		callChan:  make(chan *Call),
		shouldRun: true,
	}
	reqId    uint64
	reqMutex sync.RWMutex
	reqMap   = make(map[uint64]*Call)
	wg       sync.WaitGroup
)

// Call receives an RPC request from other pitaya servers and forward it to the
// sidecar client so that it processes them, afterwards it gets the client
// response and send it back to the callee
func (s *Sidecar) Call(ctx context.Context, req *protos.Request) (*protos.Response, error) {
	call := &Call{
		ctx:   ctx,
		req:   req,
		done:  make(chan (bool), 1),
		reqId: atomic.AddUint64(&reqId, 1),
	}

	reqMutex.Lock()
	reqMap[call.reqId] = call
	reqMutex.Unlock()

	s.callChan <- call

	defer func() {
		reqMutex.Lock()
		delete(reqMap, call.reqId)
		reqMutex.Unlock()
	}()

	select {
	case <-call.done:
		return call.res, nil
	case <-time.After(sidecar.config.GetDuration("pitaya.sidecar.calltimeout")):
		close(call.done)
		return &protos.Response{}, constants.ErrSidecarCallTimeout
	}
}

// SessionBindRemote is meant to frontend servers so its not implemented here
func (s *Sidecar) SessionBindRemote(ctx context.Context, msg *protos.BindMsg) (*protos.Response, error) {
	return nil, constants.ErrNotImplemented
}

// PushToUser is meant to frontend servers so its not implemented here
func (s *Sidecar) PushToUser(ctx context.Context, push *protos.Push) (*protos.Response, error) {
	return nil, constants.ErrNotImplemented
}

// KickUser is meant to frontend servers so its not implemented here
func (s *Sidecar) KickUser(ctx context.Context, kick *protos.KickMsg) (*protos.KickAnswer, error) {
	return nil, constants.ErrNotImplemented
}

// FinishRPC is called when the sidecar client returns the answer to an RPC
// call, after this method happens, the Call method above returns
func (s *SidecarServer) FinishRPC(ctx context.Context, res *protos.RPCResponse) {
	reqMutex.RLock()
	defer reqMutex.RUnlock()
	call, ok := reqMap[res.ReqId]
	if ok {
		call.res = res.Res
		call.err = res.Err
		close(call.done)
	}
}

// GetServer is called by the sidecar client to get the information from
// a pitaya server by passing its ID
func (s *SidecarServer) GetServer(ctx context.Context, in *protos.Server) (*protos.Server, error) {
	server, err := pitaya.GetServerByID(in.Id)
	if err != nil {
		return nil, err
	}
	res := &protos.Server{
		Id:       server.ID,
		Frontend: server.Frontend,
		Type:     server.Type,
		Metadata: server.Metadata,
		Hostname: server.Hostname,
	}
	return res, nil
}

// ListenRPC keeps a bidirectional stream open between the sidecar client and
// server, it sends incoming RPC from other pitaya servers to the client and
// also listens for incoming answers from the client. This method is the most
// important one here and is where is defined our async model.
func (s *SidecarServer) ListenRPC(stream protos.Sidecar_ListenRPCServer) error {
	go func() {
		for {
			res, err := stream.Recv()
			if err != nil {
				logger.Log.Errorf("got error in GRPC stream, %s", err)
				close(sidecar.stopChan)
				return
			}
			// TODO fix context to fix tracing
			s.FinishRPC(context.Background(), res)
		}
	}()
	for sidecar.shouldRun {
		select {
		case call := <-sidecar.callChan:
			err := stream.Send(&protos.SidecarRequest{ReqId: call.reqId, Req: call.req})
			if err != nil {
				call.err = &protos.Error{Msg: err.Error(), Code: errors.ErrBadRequestCode}
				close(call.done)
			}
		case <-sidecar.stopChan:
			sidecar.shouldRun = false
		}
	}
	logger.Log.Info("exiting sidecar ListenRPC routine because stopChan was closed")
	return nil
}

// SendRPC is called by the sidecar client when it wants to send RPC requests to
// other pitaya servers
func (s *SidecarServer) SendRPC(ctx context.Context, in *protos.RequestTo) (*protos.Response, error) {
	return pitaya.RawRPC(context.Background(), in.ServerID, in.Msg.Route, in.Msg.Data)
}

// SendPush is called by the sidecar client when it wants to send a push to an
// user through a frontend server
func (s *SidecarServer) SendPush(ctx context.Context, in *protos.PushRequest) (*protos.PushResponse, error) {
	push := in.GetPush()
	failedUids, err := pitaya.SendPushToUsers(push.Route, push.GetData(), []string{push.Uid}, in.FrontendType)
	res := &protos.PushResponse{
		FailedUids: failedUids,
	}
	if err != nil {
		res.HasFailed = true
	} else {
		res.HasFailed = false
	}
	return res, nil // can't send the error here because if we do, it will throw an exception in csharp side
}

// SendKick is called by the sidecar client when it wants to send a kick to an
// user through a frontend server
func (s *SidecarServer) SendKick(ctx context.Context, in *protos.KickRequest) (*protos.PushResponse, error) {
	failedUids, err := pitaya.SendKickToUsers([]string{in.GetKick().GetUserId()}, in.FrontendType)
	res := &protos.PushResponse{
		FailedUids: failedUids,
	}
	if err != nil {
		res.HasFailed = true
	} else {
		res.HasFailed = false
	}
	return res, nil // can't send the error here because if we do, it will throw an exception in csharp side
}

// StartPitaya instantiates a pitaya server and starts it. It must be called
// during the initialization of the sidecar client, all other methods will only
// work when this one was already called
func (s *SidecarServer) StartPitaya(ctx context.Context, req *protos.StartPitayaRequest) (*protos.Error, error) {
	config := req.GetConfig()
	pitaya.Configure(
		config.GetFrontend(),
		config.GetType(),
		pitaya.Cluster,
		config.GetMetadata(),
		sidecar.config.GetViper(),
	)

	pitaya.SetDebug(req.GetDebugLog())

	// TODO support frontend servers
	if config.GetFrontend() {
		//t := acceptor.NewTCPAcceptor(":3250") pitaya.AddAcceptor(t)
		logger.Log.Fatal("Frontend servers not supported yet")
	}

	ns, err := cluster.NewNatsRPCServer(pitaya.GetConfig(), pitaya.GetServer(), pitaya.GetMetricsReporters(), pitaya.GetDieChan())
	if err != nil {
		return nil, err
	}

	// register the sidecar as the pitaya server so that calls will be delivered
	// here and we can forward to the remote process
	ns.SetPitayaServer(sidecar)
	pitaya.SetRPCServer(ns)
	// TODO maybe we should return error in pitaya start. maybe recover from fatal
	// TODO make this method return error so that I can catch it
	go func() {
		wg.Add(1)
		defer func() {
			wg.Done()
		}()
		pitaya.Start()
	}()
	return &protos.Error{}, nil
}

// StopPitaya stops the instantiated pitaya server and must always be called
// when the client is dying so that we can correctly gracefully shutdown pitaya
func (s *SidecarServer) StopPitaya(ctx context.Context, req *emptypb.Empty) (*protos.Error, error) {
	logger.Log.Info("received stop request, will stop pitaya server")
	close(sidecar.stopChan)
	return &protos.Error{}, nil
}

func checkError(err error) {
	if err != nil {
		logger.Log.Fatalf("failed to start sidecar: %s", err)
	}
}

// StartSidecar starts the sidecar server, it instantiates the GRPC server and
// listens for incoming client connections. This is the very first method that
// is called when the sidecar is starting.
func StartSidecar(cfg *config.Config) {
	// Start our own logger
	sidecar.log = logrus.WithField("source", "sidecar")
	pitaya.SetLogger(sidecar.log)

	sidecar.config = cfg
	sidecar.sidecarServer = &SidecarServer{}

	bindAddr := cfg.GetString("pitaya.sidecar.bind")
	u, err := url.Parse(bindAddr)
	checkError(err)
	var addr string
	if u.Scheme == "unix" {
		addr = u.Path
	} else if u.Scheme == "tcp" {
		addr = u.Host
	} else {
		logger.Log.Fatal("only supported schemes are unix and tcp, review your bindaddr config")
	}

	sidecar.listener, err = net.Listen(u.Scheme, addr)
	checkError(err)
	defer sidecar.listener.Close()
	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	protos.RegisterSidecarServer(grpcServer, sidecar.sidecarServer)
	go func() {
		err = grpcServer.Serve(sidecar.listener)
		if err != nil {
			logger.Log.Errorf("error serving GRPC: %s", err)
			select {
			case <-sidecar.stopChan:
				break
			default:
				close(sidecar.stopChan)
			}
		}
	}()

	logger.Log.Infof("sidecar listening at %s", sidecar.listener.Addr())

	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)

	// stop server
	select {
	case <-sidecar.stopChan:
		logger.Log.Warn("the app will shutdown in a few seconds")
		pitaya.Shutdown()
	case s := <-sg:
		logger.Log.Warn("got signal: ", s, ", shutting down...")
		close(sidecar.stopChan)
	}

	// wait for pitaya to shutdown itself before exiting
	wg.Wait()

}
