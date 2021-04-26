package sidecar

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	pitaya "github.com/topfreegames/pitaya/pkg"
	"github.com/topfreegames/pitaya/pkg/acceptor"
	"github.com/topfreegames/pitaya/pkg/logger"
	"github.com/topfreegames/pitaya/pkg/protos"
	"google.golang.org/grpc"
)

type Sidecar struct {
	config        *viper.Viper
	sidecarServer protos.SidecarServer
	stopChan      chan bool
	log           *logrus.Entry
}

type SidecarServer struct {
	protos.UnimplementedPitayaServer
}

var (
	sidecar = &Sidecar{
		stopChan: make(chan bool),
	}
)

func (s *SidecarServer) StartPitaya(ctx context.Context, req *protos.StartPitayaRequest) (*protos.Error, error) {
	// TODO receive a valid pitaya config here?
	config := req.GetConfig()
	pitaya.Configure(
		config.GetIsFrontend(),
		config.GetServerType(),
		pitaya.Cluster,
		config.GetMetadata(),
	)

	pitaya.SetDebug(config.GetDebugLog())

	if config.GetIsFrontend() {
		// TODO configure port!
		t := acceptor.NewTCPAcceptor(":3250")
		pitaya.AddAcceptor(t)
	}

	// TODO maybe we should return error in pitaya start. maybe recover from fatal
	pitaya.Start()
	return &protos.Error{}, nil
}

func checkError(err error) {
	if err != nil {
		logger.Log.Fatalf("failed to start sidecar: %s", err)
	}
}

func StartSidecar(cfg *viper.Viper) {
	// Start our own logger
	sidecar.log = logrus.WithField("source", "sidecar")
	pitaya.SetLogger(sidecar.log)

	sidecar.config = cfg
	sidecar.sidecarServer = &SidecarServer{}

	// TODO: fix configs
	bindAddr := cfg.GetString("pitaya.sidecar.bindaddr")
	port := cfg.GetInt("pitaya.sidecar.listenport")

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bindAddr, port))
	checkError(err)
	var opts []grpc.ServerOption
	// TODO: tls shit

	grpcServer := grpc.NewServer(opts...)
	protos.RegisterSidecarServer(grpcServer, sidecar.sidecarServer)
	go func() {
		err = grpcServer.Serve(l)
		if err != nil {
			logger.Log.Errorf("error serving GRPC: %s", err)
			close(sidecar.stopChan)
		}
	}()

	logger.Log.Infof("sidecar listening at %s:%d", bindAddr, port)

	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)

	// stop server
	select {
	case <-sidecar.stopChan:
		logger.Log.Warn("the app will shutdown in a few seconds")
	case s := <-sg:
		logger.Log.Warn("got signal: ", s, ", shutting down...")
		close(sidecar.stopChan)
	}

	close(pitaya.GetDieChan())
}
