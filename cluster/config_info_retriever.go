package cluster

import "github.com/topfreegames/pitaya/v2/config"

// ConfigInfoRetriever gets cluster info from config
// Implements InfoRetriever interface
type ConfigInfoRetriever struct {
	region string
}

// NewConfigInfoRetriever returns a *ConfigInfoRetriever
func NewConfigInfoRetriever(config config.ConfigInfoRetrieverConfig) *ConfigInfoRetriever {
	return &ConfigInfoRetriever{
		region: config.Region,
	}
}

// Region gets server's region from config
func (c *ConfigInfoRetriever) Region() string {
	return c.region
}
