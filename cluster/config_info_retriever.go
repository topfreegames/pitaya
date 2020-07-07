package cluster

import "github.com/topfreegames/pitaya/config"

// ConfigInfoRetriever gets cluster info from config
// Implements InfoRetriever interface
type ConfigInfoRetriever struct {
	region string
}

// ConfigInfoRetrieverConfig provides ConfigInfoRetriever configuration
type ConfigInfoRetrieverConfig struct {
	Region string
}

// NewDefaultConfigInfoRetrieverConfig provides default configuration for ConfigInfoRetriever
func NewDefaultConfigInfoRetrieverConfig() ConfigInfoRetrieverConfig {
	return ConfigInfoRetrieverConfig{
		Region: "",
	}
}

// NewConfigInfoRetrieverConfig reads from config to build configuration for ConfigInfoRetriever
func NewConfigInfoRetrieverConfig(c *config.Config) ConfigInfoRetrieverConfig {
	return ConfigInfoRetrieverConfig{
		Region: c.GetString("pitaya.cluster.info.region"),
	}
}

// NewConfigInfoRetriever returns a *ConfigInfoRetriever
func NewConfigInfoRetriever(config ConfigInfoRetrieverConfig) *ConfigInfoRetriever {
	return &ConfigInfoRetriever{
		region: config.Region,
	}
}

// Region gets server's region from config
func (c *ConfigInfoRetriever) Region() string {
	return c.region
}
