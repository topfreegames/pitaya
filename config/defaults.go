package config

func fillDefaultValues(c Config) {
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
		"pitaya.cluster.rpc.client.nats.requesttimeout":         pitayaConfig.Cluster.RPC.Client.Nats.RequestTimeout,
		"pitaya.cluster.rpc.server.grpc.port":                   pitayaConfig.Cluster.RPC.Server.Grpc.Port,
		"pitaya.cluster.rpc.server.nats.connect":                pitayaConfig.Cluster.RPC.Server.Nats.Connect,
		"pitaya.cluster.rpc.server.nats.connectiontimeout":      pitayaConfig.Cluster.RPC.Server.Nats.ConnectionTimeout,
		"pitaya.cluster.rpc.server.nats.maxreconnectionretries": pitayaConfig.Cluster.RPC.Server.Nats.MaxReconnectionRetries,
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

	// Fix multi-source merging for viper
	viper, ok := c.(*viperImpl)

	for param := range defaultsMap {
		val := c.Get(param)
		if val == nil {
			if ok {
				viper.SetDefault(param, defaultsMap[param])
			}
		} else {
			if ok {
				viper.SetDefault(param, val)
			}
			c.Set(param, val)
		}

	}

}
