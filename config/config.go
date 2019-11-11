// Copyright (c) TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is a wrapper around a viper config
type Config struct {
	config *viper.Viper
}

// NewConfig creates a new config with a given viper config if given
func NewConfig(cfgs ...*viper.Viper) *Config {
	var cfg *viper.Viper
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	} else {
		cfg = viper.New()
	}

	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	cfg.AutomaticEnv()
	c := &Config{config: cfg}
	c.fillDefaultValues()
	return c
}

func (c *Config) fillDefaultValues() {
	defaultsMap := map[string]interface{}{
		"pitaya.buffer.agent.messages": 100,
		// the max buffer size that nats will accept, if this buffer overflows, messages will begin to be dropped
		"pitaya.buffer.cluster.rpc.server.nats.messages":        75,
		"pitaya.buffer.cluster.rpc.server.nats.push":            100,
		"pitaya.buffer.handler.localprocess":                    20,
		"pitaya.buffer.handler.remoteprocess":                   20,
		"pitaya.cluster.info.region":                            "",
		"pitaya.cluster.rpc.client.grpc.dialtimeout":            "5s",
		"pitaya.cluster.rpc.client.grpc.requesttimeout":         "5s",
		"pitaya.cluster.rpc.client.grpc.lazyconnection":         false,
		"pitaya.cluster.rpc.client.nats.connect":                "nats://localhost:4222",
		"pitaya.cluster.rpc.client.nats.connectiontimeout":      "2s",
		"pitaya.cluster.rpc.client.nats.maxreconnectionretries": 15,
		"pitaya.cluster.rpc.client.nats.requesttimeout":         "5s",
		"pitaya.cluster.rpc.server.grpc.externalport":           3434,
		"pitaya.cluster.rpc.server.grpc.port":                   3434,
		"pitaya.cluster.rpc.server.nats.connect":                "nats://localhost:4222",
		"pitaya.cluster.rpc.server.nats.connectiontimeout":      "2s",
		"pitaya.cluster.rpc.server.nats.maxreconnectionretries": 15,
		"pitaya.cluster.sd.etcd.dialtimeout":                    "5s",
		"pitaya.cluster.sd.etcd.endpoints":                      "localhost:2379",
		"pitaya.cluster.sd.etcd.grantlease.maxretries":          15,
		"pitaya.cluster.sd.etcd.grantlease.retryinterval":       "5s",
		"pitaya.cluster.sd.etcd.grantlease.timeout":             "60s",
		"pitaya.cluster.sd.etcd.heartbeat.log":                  false,
		"pitaya.cluster.sd.etcd.heartbeat.ttl":                  "60s",
		"pitaya.cluster.sd.etcd.prefix":                         "pitaya/",
		"pitaya.cluster.sd.etcd.revoke.timeout":                 "5s",
		"pitaya.cluster.sd.etcd.syncservers.interval":           "120s",
		"pitaya.cluster.sd.etcd.shutdown.delay":                 "10ms",
		"pitaya.cluster.sd.etcd.servertypeblacklist":            nil,
		"pitaya.cluster.sd.etcd.syncserversparallelism":         10,
		// the sum of this config among all the frontend servers should always be less than
		// the sum of pitaya.buffer.cluster.rpc.server.nats.messages, for covering the worst case scenario
		// a single backend server should have the config pitaya.buffer.cluster.rpc.server.nats.messages bigger
		// than the sum of the config pitaya.concurrency.handler.dispatch among all frontend servers
		"pitaya.concurrency.handler.dispatch":              25,
		"pitaya.concurrency.remote.service":                30,
		"pitaya.defaultpipelines.structvalidation.enabled": false,
		"pitaya.groups.etcd.dialtimeout":                   "5s",
		"pitaya.groups.etcd.endpoints":                     "localhost:2379",
		"pitaya.groups.etcd.prefix":                        "pitaya/",
		"pitaya.groups.etcd.transactiontimeout":            "5s",
		"pitaya.groups.memory.tickduration":                "30s",
		"pitaya.handler.messages.compression":              true,
		"pitaya.heartbeat.interval":                        "30s",
		"pitaya.metrics.additionalTags":                    map[string]string{},
		"pitaya.metrics.constTags":                         map[string]string{},
		"pitaya.metrics.custom":                            map[string]interface{}{},
		"pitaya.metrics.periodicMetrics.period":            "15s",
		"pitaya.metrics.prometheus.enabled":                false,
		"pitaya.metrics.prometheus.port":                   9090,
		"pitaya.metrics.statsd.enabled":                    false,
		"pitaya.metrics.statsd.host":                       "localhost:9125",
		"pitaya.metrics.statsd.prefix":                     "pitaya.",
		"pitaya.metrics.statsd.rate":                       1,
		"pitaya.modules.bindingstorage.etcd.dialtimeout":   "5s",
		"pitaya.modules.bindingstorage.etcd.endpoints":     "localhost:2379",
		"pitaya.modules.bindingstorage.etcd.leasettl":      "1h",
		"pitaya.modules.bindingstorage.etcd.prefix":        "pitaya/",
		"pitaya.conn.ratelimiting.limit":                   20,
		"pitaya.conn.ratelimiting.interval":                "1s",
		"pitaya.conn.ratelimiting.forcedisable":            false,
		"pitaya.session.unique":                            true,
		"pitaya.worker.concurrency":                        1,
		"pitaya.worker.redis.pool":                         "10",
		"pitaya.worker.redis.url":                          "localhost:6379",
		"pitaya.worker.retry.enabled":                      true,
		"pitaya.worker.retry.exponential":                  2,
		"pitaya.worker.retry.max":                          5,
		"pitaya.worker.retry.maxDelay":                     10,
		"pitaya.worker.retry.maxRandom":                    10,
		"pitaya.worker.retry.minDelay":                     0,
	}

	for param := range defaultsMap {
		if c.config.Get(param) == nil {
			c.config.SetDefault(param, defaultsMap[param])
		}
	}
}

// GetDuration returns a duration from the inner config
func (c *Config) GetDuration(s string) time.Duration {
	return c.config.GetDuration(s)
}

// GetString returns a string from the inner config
func (c *Config) GetString(s string) string {
	return c.config.GetString(s)
}

// GetInt returns an int from the inner config
func (c *Config) GetInt(s string) int {
	return c.config.GetInt(s)
}

// GetBool returns an boolean from the inner config
func (c *Config) GetBool(s string) bool {
	return c.config.GetBool(s)
}

// GetStringSlice returns a string slice from the inner config
func (c *Config) GetStringSlice(s string) []string {
	return c.config.GetStringSlice(s)
}

// Get returns an interface from the inner config
func (c *Config) Get(s string) interface{} {
	return c.config.Get(s)
}

// GetStringMapString returns a string map string from the inner config
func (c *Config) GetStringMapString(s string) map[string]string {
	return c.config.GetStringMapString(s)
}

// UnmarshalKey unmarshals key into v
func (c *Config) UnmarshalKey(s string, v interface{}) error {
	return c.config.UnmarshalKey(s, v)
}
