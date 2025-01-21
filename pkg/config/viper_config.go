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
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/spf13/viper"
)

// Config is a wrapper around a viper config
type Config struct {
	viper.Viper
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
	c := &Config{*cfg}
	c.fillDefaultValues()
	return c
}

func (c *Config) fillDefaultValues() {
	pitayaConfig := NewDefaultPitayaConfig()

	defaultsMap := map[string]interface{}{
		"pitaya.serializertype":        pitayaConfig.SerializerType,
		"pitaya.buffer.agent.messages": pitayaConfig.Buffer.Agent.Messages,
		// the max buffer size that nats will accept, if this buffer overflows, messages will begin to be dropped
		"pitaya.buffer.handler.localprocess":                    pitayaConfig.Buffer.Handler.LocalProcess,
		"pitaya.buffer.handler.remoteprocess":                   pitayaConfig.Buffer.Handler.RemoteProcess,
		"pitaya.cluster.info.region":                            pitayaConfig.Cluster.Info.Region,
		"pitaya.cluster.rpc.client.grpc.dialtimeout":            pitayaConfig.Cluster.RPC.Client.Grpc.DialTimeout,
		"pitaya.cluster.rpc.client.grpc.requesttimeout":         pitayaConfig.Cluster.RPC.Client.Grpc.RequestTimeout,
		"pitaya.cluster.rpc.client.grpc.lazyconnection":         pitayaConfig.Cluster.RPC.Client.Grpc.LazyConnection,
		"pitaya.cluster.rpc.client.nats.connect":                pitayaConfig.Cluster.RPC.Client.Nats.Connect,
		"pitaya.cluster.rpc.client.nats.connectiontimeout":      pitayaConfig.Cluster.RPC.Client.Nats.ConnectionTimeout,
		"pitaya.cluster.rpc.client.nats.maxreconnectionretries": pitayaConfig.Cluster.RPC.Client.Nats.MaxReconnectionRetries,
		"pitaya.cluster.rpc.client.nats.websocketcompression":   pitayaConfig.Cluster.RPC.Client.Nats.WebsocketCompression,
		"pitaya.cluster.rpc.client.nats.reconnectjitter":        pitayaConfig.Cluster.RPC.Client.Nats.ReconnectJitter,
		"pitaya.cluster.rpc.client.nats.reconnectjittertls":     pitayaConfig.Cluster.RPC.Client.Nats.ReconnectJitterTLS,
		"pitaya.cluster.rpc.client.nats.reconnectwait":          pitayaConfig.Cluster.RPC.Client.Nats.ReconnectWait,
		"pitaya.cluster.rpc.client.nats.pinginterval":           pitayaConfig.Cluster.RPC.Client.Nats.PingInterval,
		"pitaya.cluster.rpc.client.nats.maxpingsoutstanding":    pitayaConfig.Cluster.RPC.Client.Nats.MaxPingsOutstanding,
		"pitaya.cluster.rpc.client.nats.requesttimeout":         pitayaConfig.Cluster.RPC.Client.Nats.RequestTimeout,
		"pitaya.cluster.rpc.server.grpc.port":                   pitayaConfig.Cluster.RPC.Server.Grpc.Port,
		"pitaya.cluster.rpc.server.nats.connect":                pitayaConfig.Cluster.RPC.Server.Nats.Connect,
		"pitaya.cluster.rpc.server.nats.connectiontimeout":      pitayaConfig.Cluster.RPC.Server.Nats.ConnectionTimeout,
		"pitaya.cluster.rpc.server.nats.maxreconnectionretries": pitayaConfig.Cluster.RPC.Server.Nats.MaxReconnectionRetries,
		"pitaya.cluster.rpc.server.nats.websocketcompression":   pitayaConfig.Cluster.RPC.Server.Nats.WebsocketCompression,
		"pitaya.cluster.rpc.server.nats.reconnectjitter":        pitayaConfig.Cluster.RPC.Server.Nats.ReconnectJitter,
		"pitaya.cluster.rpc.server.nats.reconnectjittertls":     pitayaConfig.Cluster.RPC.Server.Nats.ReconnectJitterTLS,
		"pitaya.cluster.rpc.server.nats.reconnectwait":          pitayaConfig.Cluster.RPC.Server.Nats.ReconnectWait,
		"pitaya.cluster.rpc.server.nats.pinginterval":           pitayaConfig.Cluster.RPC.Server.Nats.PingInterval,
		"pitaya.cluster.rpc.server.nats.maxpingsoutstanding":    pitayaConfig.Cluster.RPC.Server.Nats.MaxPingsOutstanding,
		"pitaya.cluster.rpc.server.nats.services":               pitayaConfig.Cluster.RPC.Server.Nats.Services,
		"pitaya.cluster.rpc.server.nats.buffer.messages":        pitayaConfig.Cluster.RPC.Server.Nats.Buffer.Messages,
		"pitaya.cluster.rpc.server.nats.buffer.push":            pitayaConfig.Cluster.RPC.Server.Nats.Buffer.Push,
		"pitaya.cluster.sd.etcd.dialtimeout":                    pitayaConfig.Cluster.SD.Etcd.DialTimeout,
		"pitaya.cluster.sd.etcd.endpoints":                      pitayaConfig.Cluster.SD.Etcd.Endpoints,
		"pitaya.cluster.sd.etcd.prefix":                         pitayaConfig.Cluster.SD.Etcd.Prefix,
		"pitaya.cluster.sd.etcd.grantlease.maxretries":          pitayaConfig.Cluster.SD.Etcd.GrantLease.MaxRetries,
		"pitaya.cluster.sd.etcd.grantlease.retryinterval":       pitayaConfig.Cluster.SD.Etcd.GrantLease.RetryInterval,
		"pitaya.cluster.sd.etcd.grantlease.timeout":             pitayaConfig.Cluster.SD.Etcd.GrantLease.Timeout,
		"pitaya.cluster.sd.etcd.heartbeat.log":                  pitayaConfig.Cluster.SD.Etcd.Heartbeat.Log,
		"pitaya.cluster.sd.etcd.heartbeat.ttl":                  pitayaConfig.Cluster.SD.Etcd.Heartbeat.TTL,
		"pitaya.cluster.sd.etcd.revoke.timeout":                 pitayaConfig.Cluster.SD.Etcd.Revoke.Timeout,
		"pitaya.cluster.sd.etcd.syncservers.interval":           pitayaConfig.Cluster.SD.Etcd.SyncServers.Interval,
		"pitaya.cluster.sd.etcd.syncservers.parallelism":        pitayaConfig.Cluster.SD.Etcd.SyncServers.Parallelism,
		"pitaya.cluster.sd.etcd.shutdown.delay":                 pitayaConfig.Cluster.SD.Etcd.Shutdown.Delay,
		"pitaya.cluster.sd.etcd.servertypeblacklist":            pitayaConfig.Cluster.SD.Etcd.ServerTypesBlacklist,
		// the sum of this config among all the frontend servers should always be less than
		// the sum of pitaya.buffer.cluster.rpc.server.nats.messages, for covering the worst case scenario
		// a single backend server should have the config pitaya.buffer.cluster.rpc.server.nats.messages bigger
		// than the sum of the config pitaya.concurrency.handler.dispatch among all frontend servers
		"pitaya.acceptor.proxyprotocol":                    pitayaConfig.Acceptor.ProxyProtocol,
		"pitaya.concurrency.handler.dispatch":              pitayaConfig.Concurrency.Handler.Dispatch,
		"pitaya.defaultpipelines.structvalidation.enabled": pitayaConfig.DefaultPipelines.StructValidation.Enabled,
		"pitaya.groups.etcd.dialtimeout":                   pitayaConfig.Groups.Etcd.DialTimeout,
		"pitaya.groups.etcd.endpoints":                     pitayaConfig.Groups.Etcd.Endpoints,
		"pitaya.groups.etcd.prefix":                        pitayaConfig.Groups.Etcd.Prefix,
		"pitaya.groups.etcd.transactiontimeout":            pitayaConfig.Groups.Etcd.TransactionTimeout,
		"pitaya.groups.memory.tickduration":                pitayaConfig.Groups.Memory.TickDuration,
		"pitaya.handler.messages.compression":              pitayaConfig.Handler.Messages.Compression,
		"pitaya.heartbeat.interval":                        pitayaConfig.Heartbeat.Interval,
		"pitaya.metrics.additionalLabels":                  pitayaConfig.Metrics.AdditionalLabels,
		"pitaya.metrics.constLabels":                       pitayaConfig.Metrics.ConstLabels,
		"pitaya.metrics.custom":                            pitayaConfig.Metrics.Custom,
		"pitaya.metrics.period":                            pitayaConfig.Metrics.Period,
		"pitaya.metrics.prometheus.enabled":                pitayaConfig.Metrics.Prometheus.Enabled,
		"pitaya.metrics.prometheus.port":                   pitayaConfig.Metrics.Prometheus.Port,
		"pitaya.metrics.statsd.enabled":                    pitayaConfig.Metrics.Statsd.Enabled,
		"pitaya.metrics.statsd.host":                       pitayaConfig.Metrics.Statsd.Host,
		"pitaya.metrics.statsd.prefix":                     pitayaConfig.Metrics.Statsd.Prefix,
		"pitaya.metrics.statsd.rate":                       pitayaConfig.Metrics.Statsd.Rate,
		"pitaya.modules.bindingstorage.etcd.dialtimeout":   pitayaConfig.Modules.BindingStorage.Etcd.DialTimeout,
		"pitaya.modules.bindingstorage.etcd.endpoints":     pitayaConfig.Modules.BindingStorage.Etcd.Endpoints,
		"pitaya.modules.bindingstorage.etcd.leasettl":      pitayaConfig.Modules.BindingStorage.Etcd.LeaseTTL,
		"pitaya.modules.bindingstorage.etcd.prefix":        pitayaConfig.Modules.BindingStorage.Etcd.Prefix,
		"pitaya.conn.ratelimiting.limit":                   pitayaConfig.Conn.RateLimiting.Limit,
		"pitaya.conn.ratelimiting.interval":                pitayaConfig.Conn.RateLimiting.Interval,
		"pitaya.conn.ratelimiting.forcedisable":            pitayaConfig.Conn.RateLimiting.ForceDisable,
		"pitaya.session.unique":                            pitayaConfig.Session.Unique,
		"pitaya.session.drain.enabled":                     pitayaConfig.Session.Drain.Enabled,
		"pitaya.session.drain.timeout":                     pitayaConfig.Session.Drain.Timeout,
		"pitaya.session.drain.period":                      pitayaConfig.Session.Drain.Period,
		"pitaya.worker.concurrency":                        pitayaConfig.Worker.Concurrency,
		"pitaya.worker.redis.pool":                         pitayaConfig.Worker.Redis.Pool,
		"pitaya.worker.redis.url":                          pitayaConfig.Worker.Redis.ServerURL,
		"pitaya.worker.retry.enabled":                      pitayaConfig.Worker.Retry.Enabled,
		"pitaya.worker.retry.exponential":                  pitayaConfig.Worker.Retry.Exponential,
		"pitaya.worker.retry.max":                          pitayaConfig.Worker.Retry.Max,
		"pitaya.worker.retry.maxDelay":                     pitayaConfig.Worker.Retry.MaxDelay,
		"pitaya.worker.retry.maxRandom":                    pitayaConfig.Worker.Retry.MaxRandom,
		"pitaya.worker.retry.minDelay":                     pitayaConfig.Worker.Retry.MinDelay,
	}

	for param := range defaultsMap {
		val := c.Get(param)
		if val == nil {
			c.SetDefault(param, defaultsMap[param])
		} else {
			c.SetDefault(param, val)
			c.Set(param, val)
		}

	}
}

// UnmarshalKey unmarshals key into v
func (c *Config) UnmarshalKey(key string, rawVal interface{}) error {
	key = strings.ToLower(key)
	delimiter := "."
	prefix := key + delimiter

	i := c.Get(key)
	if i == nil {
		return nil
	}
	if isStringMapInterface(i) {
		val := i.(map[string]interface{})
		keys := c.AllKeys()
		for _, k := range keys {
			if !strings.HasPrefix(k, prefix) {
				continue
			}
			mk := strings.TrimPrefix(k, prefix)
			mk = strings.Split(mk, delimiter)[0]
			if _, exists := val[mk]; exists {
				continue
			}
			mv := c.Get(key + delimiter + mk)
			if mv == nil {
				continue
			}
			val[mk] = mv
		}
		i = val
	}
	return decode(i, defaultDecoderConfig(rawVal))
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
