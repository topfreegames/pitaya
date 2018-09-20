package cluster

import "github.com/topfreegames/pitaya/config"

// ConfigInfoRetriever gets cluster info from config
// Implements InfoRetriever interface
type ConfigInfoRetriever struct {
	region string
}

// NewConfigInfoRetriever returns a *ConfigInfoRetriever
func NewConfigInfoRetriever(c *config.Config) *ConfigInfoRetriever {
	return &ConfigInfoRetriever{
		region: c.GetString("pitaya.cluster.info.region"),
	}
}

// Region gets server's region from config
func (c *ConfigInfoRetriever) Region() string {
	return c.region
}
