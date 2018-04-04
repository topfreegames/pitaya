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
	"fmt"
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
		"pitaya.buffer.agent.messages":                  16,
		"pitaya.buffer.cluster.rpc.server.messages":     1000,
		"pitaya.buffer.cluster.rpc.server.push":         100,
		"pitaya.buffer.handler.localprocess":            10,
		"pitaya.buffer.handler.remoteprocess":           10,
		"pitaya.concurrency.handler.dispatch":           10,
		"pitaya.concurrency.remote.service":             10,
		"pitaya.cluster.rpc.client.nats.connect":        "nats://localhost:4222",
		"pitaya.cluster.rpc.client.nats.requesttimeout": "5s",
		"pitaya.cluster.rpc.server.nats.connect":        "nats://localhost:4222",
		"pitaya.cluster.sd.etcd.dialtimeout":            "5s",
		"pitaya.cluster.sd.etcd.endpoints":              "localhost:2379",
		"pitaya.cluster.sd.etcd.prefix":                 "pitaya/",
		"pitaya.cluster.sd.etcd.heartbeat.interval":     "20s",
		"pitaya.cluster.sd.etcd.heartbeat.ttl":          "60s",
		"pitaya.cluster.sd.etcd.syncservers.interval":   "120s",
		"pitaya.heartbeat.interval":                     "30s",
	}

	for param := range defaultsMap {
		if c.config.Get(param) == nil {
			c.config.SetDefault(param, defaultsMap[param])
		}
	}
}

// GetConcurrency retrieves concurrency config for a given suffix
func (c *Config) GetConcurrency(s string) int {
	return c.config.GetInt(fmt.Sprintf("pitaya.concurrency.%s", s))
}

// GetBuffer retrieves buffer config for a given suffix
func (c *Config) GetBuffer(s string) int {
	return c.config.GetInt(fmt.Sprintf("pitaya.buffer.%s", s))
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

// GetStringSlice returns a string slice from the inner config
func (c *Config) GetStringSlice(s string) []string {
	return c.config.GetStringSlice(s)
}

// Get returns an interface from the inner config
func (c *Config) Get(s string) interface{} {
	return c.config.Get(s)
}
