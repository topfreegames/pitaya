package cluster

import (
	"testing"

	"github.com/long12310225/pitaya/v2/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestInfoRetrieverRegion(t *testing.T) {
	t.Parallel()

	c := viper.New()
	c.Set("pitaya.cluster.info.region", "us")
	conf := config.NewConfig(c)

	infoRetriever := NewInfoRetriever(*config.NewInfoRetrieverConfig(conf))

	assert.Equal(t, "us", infoRetriever.Region())
}
