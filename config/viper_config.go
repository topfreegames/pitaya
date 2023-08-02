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
	"github.com/mitchellh/mapstructure"
	"reflect"
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
	customMetricsSpec := NewDefaultCustomMetricsSpec()
	builderConfig := NewDefaultBuilderConfig()
	pitayaConfig := NewDefaultPitayaConfig()
	prometheusConfig := NewDefaultPrometheusConfig()
	statsdConfig := NewDefaultStatsdConfig()
	etcdSDConfig := NewDefaultEtcdServiceDiscoveryConfig()
	natsRPCServerConfig := NewDefaultNatsRPCServerConfig()
	natsRPCClientConfig := NewDefaultNatsRPCClientConfig()
	grpcRPCClientConfig := NewDefaultGRPCClientConfig()
	grpcRPCServerConfig := NewDefaultGRPCServerConfig()
	workerConfig := NewDefaultWorkerConfig()
	enqueueOpts := NewDefaultEnqueueOpts()
	groupServiceConfig := NewDefaultMemoryGroupConfig()
	etcdGroupServiceConfig := NewDefaultEtcdGroupServiceConfig()
	rateLimitingConfig := NewDefaultRateLimitingConfig()
	infoRetrieverConfig := NewDefaultInfoRetrieverConfig()
	etcdBindingConfig := NewDefaultETCDBindingConfig()

	defaultsMap := map[string]interface{}{
		"pitaya.buffer.agent.messages": pitayaConfig.Buffer.Agent.Messages,
		// the max buffer size that nats will accept, if this buffer overflows, messages will begin to be dropped
		"pitaya.buffer.handler.localprocess":                    pitayaConfig.Buffer.Handler.LocalProcess,
		"pitaya.buffer.handler.remoteprocess":                   pitayaConfig.Buffer.Handler.RemoteProcess,
		"pitaya.cluster.info.region":                            infoRetrieverConfig.Region,
		"pitaya.cluster.rpc.client.grpc.dialtimeout":            grpcRPCClientConfig.DialTimeout,
		"pitaya.cluster.rpc.client.grpc.requesttimeout":         grpcRPCClientConfig.RequestTimeout,
		"pitaya.cluster.rpc.client.grpc.lazyconnection":         grpcRPCClientConfig.LazyConnection,
		"pitaya.cluster.rpc.client.nats.connect":                natsRPCClientConfig.Connect,
		"pitaya.cluster.rpc.client.nats.connectiontimeout":      natsRPCClientConfig.ConnectionTimeout,
		"pitaya.cluster.rpc.client.nats.maxreconnectionretries": natsRPCClientConfig.MaxReconnectionRetries,
		"pitaya.cluster.rpc.client.nats.requesttimeout":         natsRPCClientConfig.RequestTimeout,
		"pitaya.cluster.rpc.server.grpc.port":                   grpcRPCServerConfig.Port,
		"pitaya.cluster.rpc.server.nats.connect":                natsRPCServerConfig.Connect,
		"pitaya.cluster.rpc.server.nats.connectiontimeout":      natsRPCServerConfig.ConnectionTimeout,
		"pitaya.cluster.rpc.server.nats.maxreconnectionretries": natsRPCServerConfig.MaxReconnectionRetries,
		"pitaya.cluster.rpc.server.nats.services":               natsRPCServerConfig.Services,
		"pitaya.cluster.rpc.server.nats.buffer.messages":        natsRPCServerConfig.Buffer.Messages,
		"pitaya.cluster.rpc.server.nats.buffer.push":            natsRPCServerConfig.Buffer.Push,
		"pitaya.cluster.sd.etcd.dialtimeout":                    etcdSDConfig.DialTimeout,
		"pitaya.cluster.sd.etcd.endpoints":                      etcdSDConfig.Endpoints,
		"pitaya.cluster.sd.etcd.prefix":                         etcdSDConfig.Prefix,
		"pitaya.cluster.sd.etcd.grantlease.maxretries":          etcdSDConfig.GrantLease.MaxRetries,
		"pitaya.cluster.sd.etcd.grantlease.retryinterval":       etcdSDConfig.GrantLease.RetryInterval,
		"pitaya.cluster.sd.etcd.grantlease.timeout":             etcdSDConfig.GrantLease.Timeout,
		"pitaya.cluster.sd.etcd.heartbeat.log":                  etcdSDConfig.Heartbeat.Log,
		"pitaya.cluster.sd.etcd.heartbeat.ttl":                  etcdSDConfig.Heartbeat.TTL,
		"pitaya.cluster.sd.etcd.revoke.timeout":                 etcdSDConfig.Revoke.Timeout,
		"pitaya.cluster.sd.etcd.syncservers.interval":           etcdSDConfig.SyncServers.Interval,
		"pitaya.cluster.sd.etcd.syncserversparallelism":         etcdSDConfig.SyncServers.Parallelism,
		"pitaya.cluster.sd.etcd.shutdown.delay":                 etcdSDConfig.Shutdown.Delay,
		"pitaya.cluster.sd.etcd.servertypeblacklist":            etcdSDConfig.ServerTypesBlacklist,
		// the sum of this config among all the frontend servers should always be less than
		// the sum of pitaya.buffer.cluster.rpc.server.nats.messages, for covering the worst case scenario
		// a single backend server should have the config pitaya.buffer.cluster.rpc.server.nats.messages bigger
		// than the sum of the config pitaya.concurrency.handler.dispatch among all frontend servers
		"pitaya.acceptor.proxyprotocol":                    pitayaConfig.Acceptor.ProxyProtocol,
		"pitaya.concurrency.handler.dispatch":              pitayaConfig.Concurrency.Handler.Dispatch,
		"pitaya.defaultpipelines.structvalidation.enabled": builderConfig.DefaultPipelines.StructValidation.Enabled,
		"pitaya.groups.etcd.dialtimeout":                   etcdGroupServiceConfig.DialTimeout,
		"pitaya.groups.etcd.endpoints":                     etcdGroupServiceConfig.Endpoints,
		"pitaya.groups.etcd.prefix":                        etcdGroupServiceConfig.Prefix,
		"pitaya.groups.etcd.transactiontimeout":            etcdGroupServiceConfig.TransactionTimeout,
		"pitaya.groups.memory.tickduration":                groupServiceConfig.TickDuration,
		"pitaya.handler.messages.compression":              pitayaConfig.Handler.Messages.Compression,
		"pitaya.heartbeat.interval":                        pitayaConfig.Heartbeat.Interval,
		"pitaya.metrics.prometheus.additionalTags":         prometheusConfig.Prometheus.AdditionalLabels,
		"pitaya.metrics.constTags":                         prometheusConfig.ConstLabels,
		"pitaya.metrics.custom":                            customMetricsSpec,
		"pitaya.metrics.periodicMetrics.period":            pitayaConfig.Metrics.Period,
		"pitaya.metrics.prometheus.enabled":                builderConfig.Metrics.Prometheus.Enabled,
		"pitaya.metrics.prometheus.port":                   prometheusConfig.Prometheus.Port,
		"pitaya.metrics.statsd.enabled":                    builderConfig.Metrics.Statsd.Enabled,
		"pitaya.metrics.statsd.host":                       statsdConfig.Statsd.Host,
		"pitaya.metrics.statsd.prefix":                     statsdConfig.Statsd.Prefix,
		"pitaya.metrics.statsd.rate":                       statsdConfig.Statsd.Rate,
		"pitaya.modules.bindingstorage.etcd.dialtimeout":   etcdBindingConfig.DialTimeout,
		"pitaya.modules.bindingstorage.etcd.endpoints":     etcdBindingConfig.Endpoints,
		"pitaya.modules.bindingstorage.etcd.leasettl":      etcdBindingConfig.LeaseTTL,
		"pitaya.modules.bindingstorage.etcd.prefix":        etcdBindingConfig.Prefix,
		"pitaya.conn.ratelimiting.limit":                   rateLimitingConfig.Limit,
		"pitaya.conn.ratelimiting.interval":                rateLimitingConfig.Interval,
		"pitaya.conn.ratelimiting.forcedisable":            rateLimitingConfig.ForceDisable,
		"pitaya.session.unique":                            pitayaConfig.Session.Unique,
		"pitaya.worker.concurrency":                        workerConfig.Concurrency,
		"pitaya.worker.redis.pool":                         workerConfig.Redis.Pool,
		"pitaya.worker.redis.url":                          workerConfig.Redis.ServerURL,
		"pitaya.worker.retry.enabled":                      enqueueOpts.Enabled,
		"pitaya.worker.retry.exponential":                  enqueueOpts.Exponential,
		"pitaya.worker.retry.max":                          enqueueOpts.Max,
		"pitaya.worker.retry.maxDelay":                     enqueueOpts.MaxDelay,
		"pitaya.worker.retry.maxRandom":                    enqueueOpts.MaxRandom,
		"pitaya.worker.retry.minDelay":                     enqueueOpts.MinDelay,
	}

	for param := range defaultsMap {
		val := c.config.Get(param)
		if val == nil {
			c.config.SetDefault(param, defaultsMap[param])
		} else {
			c.config.SetDefault(param, val)
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

// Unmarshal unmarshals config into v
func (c *Config) Unmarshal(v interface{}) error {
	return c.config.Unmarshal(v)
}

// UnmarshalKey unmarshals key into v
func (c *Config) UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	key = strings.ToLower(key)
	delimiter := "."
	prefix := key + delimiter

	i := c.config.Get(key)
	if isStringMapInterface(i) {
		val := i.(map[string]interface{})
		keys := c.config.AllKeys()
		for _, k := range keys {
			if !strings.HasPrefix(k, prefix) {
				continue
			}
			fmt.Printf("prefix: %v\n", prefix)
			mk := strings.TrimPrefix(k, prefix)
			fmt.Printf("got key1: %v\n", mk)
			mk = strings.Split(mk, delimiter)[0]
			fmt.Printf("got key2: %v\n", mk)
			if _, exists := val[mk]; exists {
				continue
			}
			mv := c.Get(key + delimiter + mk)
			fmt.Printf("got key5: %v\n", mv)
			if mv == nil {
				continue
			}
			val[mk] = mv
		}
		i = val
	}
	return decode(i, defaultDecoderConfig(rawVal, opts...))
}

func isStringMapInterface(val interface{}) bool {
	vt := reflect.TypeOf(val)
	return vt.Kind() == reflect.Map &&
		vt.Key().Kind() == reflect.String &&
		vt.Elem().Kind() == reflect.Interface
}

// A wrapper around mapstructure.Decode that mimics the WeakDecode functionality
func decode(input interface{}, config *mapstructure.DecoderConfig) error {
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

// defaultDecoderConfig returns default mapstructure.DecoderConfig with support
// of time.Duration values & string slices
func defaultDecoderConfig(output interface{}, opts ...viper.DecoderConfigOption) *mapstructure.DecoderConfig {
	c := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c

}
