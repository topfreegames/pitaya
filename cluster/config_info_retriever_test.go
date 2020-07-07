package cluster

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/config"
)

func TestConfigInfoRetrieverRegion(t *testing.T) {
	t.Parallel()

	c := viper.New()
	c.Set("pitaya.cluster.info.region", "us")
	config := config.NewConfig(c)

	infoRetriever := NewConfigInfoRetriever(NewConfigInfoRetrieverConfig(config))

	assert.Equal(t, "us", infoRetriever.Region())
}
