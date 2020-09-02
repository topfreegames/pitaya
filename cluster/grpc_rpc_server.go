// Copyright (c) TFG Co. All Rights Reserved.
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

package cluster

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/topfreegames/pitaya/v2/config"
	"github.com/topfreegames/pitaya/v2/metrics"
	"github.com/topfreegames/pitaya/v2/protos"
)

// GRPCServer rpc server struct
type GRPCServer struct {
	server           *Server
	port             int
	metricsReporters []metrics.Reporter
	grpcSv           *grpc.Server
	pitayaServer     protos.PitayaServer
}

// GRPCServerConfig provides configuration for GRPCServer
type GRPCServerConfig struct {
	Port int
}

// NewDefaultGRPCServerConfig returns a default GRPCServerConfig
func NewDefaultGRPCServerConfig() GRPCServerConfig {
	return GRPCServerConfig{
		Port: 3434,
	}
}

// NewGRPCServerConfig reads from config to build GRPCServerConfig
func NewGRPCServerConfig(config *config.Config) GRPCServerConfig {
	return GRPCServerConfig{
		Port: config.GetInt("pitaya.cluster.rpc.server.grpc.port"),
	}
}

// NewGRPCServer constructor
func NewGRPCServer(config GRPCServerConfig, server *Server, metricsReporters []metrics.Reporter) (*GRPCServer, error) {
	gs := &GRPCServer{
		port:             config.Port,
		server:           server,
		metricsReporters: metricsReporters,
	}
	return gs, nil
}

// Init inits grpc rpc server
func (gs *GRPCServer) Init() error {
	port := gs.port
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	gs.grpcSv = grpc.NewServer()
	protos.RegisterPitayaServer(gs.grpcSv, gs.pitayaServer)
	go gs.grpcSv.Serve(lis)
	return nil
}

// SetPitayaServer sets the pitaya server
func (gs *GRPCServer) SetPitayaServer(ps protos.PitayaServer) {
	gs.pitayaServer = ps
}

// AfterInit runs after initialization
func (gs *GRPCServer) AfterInit() {}

// BeforeShutdown runs before shutdown
func (gs *GRPCServer) BeforeShutdown() {}

// Shutdown stops grpc rpc server
func (gs *GRPCServer) Shutdown() error {
	// graceful: stops the server from accepting new connections and RPCs and
	// blocks until all the pending RPCs are finished.
	// source: https://godoc.org/google.golang.org/grpc#Server.GracefulStop
	gs.grpcSv.GracefulStop()
	return nil
}
