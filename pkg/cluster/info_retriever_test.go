package cluster

import (
	config2 "github.com/topfreegames/pitaya/v2/pkg/config"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestInfoRetrieverRegion(t *testing.T) {
	t.Parallel()

	c := viper.New()
	c.Set("pitaya.cluster.info.region", "us")
	conf := config2.NewConfig(c)

	infoRetriever := NewInfoRetriever(*config2.NewInfoRetrieverConfig(conf))

	assert.Equal(t, "us", infoRetriever.Region())
}
