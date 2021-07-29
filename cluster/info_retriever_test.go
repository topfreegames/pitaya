package cluster

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v2/config"
)

func TestInfoRetrieverRegion(t *testing.T) {
	t.Parallel()

	c := viper.New()
	c.Set("pitaya.cluster.info.region", "us")
	conf := config.NewConfig(c)

	infoRetriever := NewInfoRetriever(*config.NewInfoRetrieverConfig(conf))

	assert.Equal(t, "us", infoRetriever.Region())
}
