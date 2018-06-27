package cluster

import (
	"fmt"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/spf13/viper"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/helpers"
	"github.com/topfreegames/pitaya/metrics"
	protosmocks "github.com/topfreegames/pitaya/protos/mocks"
)

func TestNewGRPCServer(t *testing.T) {
	t.Parallel()
	conf := getConfig()
	sv := getServer()
	gs, err := NewGRPCServer(conf, sv, []metrics.Reporter{})
	assert.NoError(t, err)
	assert.NotNil(t, gs)
}

func TestGRPCServerInit(t *testing.T) {
	t.Parallel()
	c := viper.New()
	p := helpers.GetFreePort(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockPitayaServer := protosmocks.NewMockPitayaServer(ctrl)

	c.Set("pitaya.cluster.rpc.server.grpc.port", p)
	conf := getConfig(c)
	sv := getServer()
	gs, err := NewGRPCServer(conf, sv, []metrics.Reporter{})
	gs.SetPitayaServer(mockPitayaServer)
	err = gs.Init()
	assert.NoError(t, err)
	assert.NotNil(t, gs)

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", p))
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, gs.grpcSv)
}
