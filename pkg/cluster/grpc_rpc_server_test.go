package cluster

import (
	"fmt"
	"github.com/topfreegames/pitaya/v2/pkg/config"
	"net"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v2/pkg/helpers"
	"github.com/topfreegames/pitaya/v2/pkg/metrics"
	protosmocks "github.com/topfreegames/pitaya/v2/pkg/protos/mocks"
)

func TestNewGRPCServer(t *testing.T) {
	t.Parallel()
	sv := getServer()
	gs, err := NewGRPCServer(*config.NewDefaultGRPCServerConfig(), sv, []metrics.Reporter{})
	assert.NoError(t, err)
	assert.NotNil(t, gs)
}

func TestGRPCServerInit(t *testing.T) {
	t.Parallel()
	c := config.NewDefaultGRPCServerConfig()
	c.Port = helpers.GetFreePort(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockPitayaServer := protosmocks.NewMockPitayaServer(ctrl)

	sv := getServer()
	gs, err := NewGRPCServer(*c, sv, []metrics.Reporter{})
	gs.SetPitayaServer(mockPitayaServer)

	tSv := gs.GetPitayaServer()
	assert.Equal(t, mockPitayaServer, tSv)
	err = gs.Init()
	assert.NoError(t, err)
	assert.NotNil(t, gs)

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", c.Port))
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, gs.grpcSv)
}
