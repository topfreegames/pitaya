package cluster

import "github.com/topfreegames/pitaya/v3/pkg/config"

// infoRetriever gets cluster info from config
// Implements InfoRetriever interface
type infoRetriever struct {
	region string
}

// NewInfoRetriever returns a *infoRetriever
func NewInfoRetriever(config config.InfoRetrieverConfig) InfoRetriever {
	return &infoRetriever{
		region: config.Region,
	}
}

// Region gets server's region from config
func (c *infoRetriever) Region() string {
	return c.region
}
