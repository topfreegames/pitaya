package config

import (
	"strconv"
	"time"

	"github.com/topfreegames/pitaya/v2/metrics/models"
)

// PitayaConfig provides configuration for a pitaya app
type PitayaConfig struct {
	HearbeatInterval           time.Duration
	MessageCompression         bool
	BufferAgentMessages        int
	BufferHandlerLocalProcess  int
	BufferHandlerRemoteProcess int
	ConcurrencyHandlerDispatch int
	SessionUnique              bool
	MetricsPeriod              time.Duration
}

// NewDefaultPitayaConfig provides default configuration for Pitaya App
func NewDefaultPitayaConfig() PitayaConfig {
	return PitayaConfig{
		HearbeatInterval:           time.Duration(30 * time.Second),
		MessageCompression:         true,
		BufferAgentMessages:        100,
		BufferHandlerLocalProcess:  20,
		BufferHandlerRemoteProcess: 20,
		ConcurrencyHandlerDispatch: 25,
		SessionUnique:              true,
		MetricsPeriod:              time.Duration(15 * time.Second),
	}
}

// NewPitayaConfig returns a config instance with values extracted from default config paths
func NewPitayaConfig(config *Config) PitayaConfig {
	return PitayaConfig{
		HearbeatInterval:           config.GetDuration("pitaya.heartbeat.interval"),
		MessageCompression:         config.GetBool("pitaya.handler.messages.compression"),
		BufferAgentMessages:        config.GetInt("pitaya.buffer.agent.messages"),
		BufferHandlerLocalProcess:  config.GetInt("pitaya.buffer.handler.localprocess"),
		BufferHandlerRemoteProcess: config.GetInt("pitaya.buffer.handler.remoteprocess"),
		ConcurrencyHandlerDispatch: config.GetInt("pitaya.concurrency.handler.dispatch"),
		SessionUnique:              config.GetBool("pitaya.session.unique"),
		MetricsPeriod:              config.GetDuration("pitaya.metrics.periodicMetrics.period"),
	}
}

// BuilderConfig provides configuration for Builder
type BuilderConfig struct {
	PitayaConfig             PitayaConfig
	IsPrometheusEnabled      bool
	IsStatsdEnabled          bool
	IsDefaultPipelineEnabled bool
}

// NewDefaultBuilderConfig provides default builder configuration
func NewDefaultBuilderConfig() BuilderConfig {
	return BuilderConfig{
		PitayaConfig:             NewDefaultPitayaConfig(),
		IsPrometheusEnabled:      false,
		IsStatsdEnabled:          false,
		IsDefaultPipelineEnabled: false,
	}
}

// NewBuilderConfig reads from config to build builder configuration
func NewBuilderConfig(config *Config) BuilderConfig {
	return BuilderConfig{
		PitayaConfig:             NewPitayaConfig(config),
		IsPrometheusEnabled:      config.GetBool("pitaya.metrics.prometheus.enabled"),
		IsStatsdEnabled:          config.GetBool("pitaya.metrics.statsd.enabled"),
		IsDefaultPipelineEnabled: config.GetBool("pitaya.defaultpipelines.structvalidation.enabled"),
	}
}

// GRPCClientConfig rpc client config struct
type GRPCClientConfig struct {
	DialTimeout    time.Duration
	LazyConnection bool
	RequestTimeout time.Duration
}

// NewDefaultGRPCClientConfig rpc client default config struct
func NewDefaultGRPCClientConfig() GRPCClientConfig {
	return GRPCClientConfig{
		DialTimeout:    time.Duration(5 * time.Second),
		LazyConnection: false,
		RequestTimeout: time.Duration(5 * time.Second),
	}
}

// NewGRPCClientConfig reads from config to build GRPCCLientConfig
func NewGRPCClientConfig(config *Config) GRPCClientConfig {
	return GRPCClientConfig{
		DialTimeout:    config.GetDuration("pitaya.cluster.rpc.client.grpc.dialtimeout"),
		LazyConnection: config.GetBool("pitaya.cluster.rpc.client.grpc.lazyconnection"),
		RequestTimeout: config.GetDuration("pitaya.cluster.rpc.client.grpc.requesttimeout"),
	}
}

// GRPCServerConfig provides configuration for GRPCServer
type GRPCServerConfig struct {
	Port int
}

// NewDefaultGRPCServerConfig returns a default GRPCServerConfig
func NewDefaultGRPCServerConfig() GRPCServerConfig {
	return GRPCServerConfig{
		Port: 3434,
	}
}

// NewGRPCServerConfig reads from config to build GRPCServerConfig
func NewGRPCServerConfig(config *Config) GRPCServerConfig {
	return GRPCServerConfig{
		Port: config.GetInt("pitaya.cluster.rpc.server.grpc.port"),
	}
}

// NatsRPCClientConfig provides nats client configuration
type NatsRPCClientConfig struct {
	Connect                string
	MaxReconnectionRetries int
	RequestTimeout         time.Duration
	ConnectionTimeout      time.Duration
}

// NewDefaultNatsRPCClientConfig provides default nats client configuration
func NewDefaultNatsRPCClientConfig() NatsRPCClientConfig {
	return NatsRPCClientConfig{
		Connect:                "nats://localhost:4222",
		MaxReconnectionRetries: 15,
		RequestTimeout:         time.Duration(5 * time.Second),
		ConnectionTimeout:      time.Duration(2 * time.Second),
	}
}

// NewNatsRPCClientConfig reads from config to build nats client configuration
func NewNatsRPCClientConfig(config *Config) NatsRPCClientConfig {
	return NatsRPCClientConfig{
		Connect:                config.GetString("pitaya.cluster.rpc.client.nats.connect"),
		MaxReconnectionRetries: config.GetInt("pitaya.cluster.rpc.client.nats.maxreconnectionretries"),
		RequestTimeout:         config.GetDuration("pitaya.cluster.rpc.client.nats.requesttimeout"),
		ConnectionTimeout:      config.GetDuration("pitaya.cluster.rpc.client.nats.connectiontimeout"),
	}
}

// NatsRPCServerConfig provides nats server configuration
type NatsRPCServerConfig struct {
	Connect                string
	MaxReconnectionRetries int
	Messages               int
	Push                   int
	Service                int
	ConnectionTimeout      time.Duration
}

// NewDefaultNatsRPCServerConfig provides default nats server configuration
func NewDefaultNatsRPCServerConfig() NatsRPCServerConfig {
	return NatsRPCServerConfig{
		Connect:                "nats://localhost:4222",
		MaxReconnectionRetries: 15,
		Messages:               75,
		Push:                   100,
		Service:                30,
		ConnectionTimeout:      time.Duration(2 * time.Second),
	}
}

// NewNatsRPCServerConfig reads from config to build nats server configuration
func NewNatsRPCServerConfig(config *Config) NatsRPCServerConfig {
	return NatsRPCServerConfig{
		Connect:                config.GetString("pitaya.cluster.rpc.server.nats.connect"),
		MaxReconnectionRetries: config.GetInt("pitaya.cluster.rpc.server.nats.maxreconnectionretries"),
		Messages:               config.GetInt("pitaya.buffer.cluster.rpc.server.nats.messages"),
		Push:                   config.GetInt("pitaya.buffer.cluster.rpc.server.nats.push"),
		Service:                config.GetInt("pitaya.concurrency.remote.service"),
		ConnectionTimeout:      config.GetDuration("pitaya.cluster.rpc.server.nats.connectiontimeout"),
	}
}

// InfoRetrieverConfig provides InfoRetriever configuration
type InfoRetrieverConfig struct {
	Region string
}

// NewDefaultInfoRetrieverConfig provides default configuration for InfoRetriever
func NewDefaultInfoRetrieverConfig() InfoRetrieverConfig {
	return InfoRetrieverConfig{
		Region: "",
	}
}

// NewInfoRetrieverConfig reads from config to build configuration for InfoRetriever
func NewInfoRetrieverConfig(c *Config) InfoRetrieverConfig {
	return InfoRetrieverConfig{
		Region: c.GetString("pitaya.cluster.info.region"),
	}
}

// EtcdServiceDiscoveryConfig Etcd service discovery config
type EtcdServiceDiscoveryConfig struct {
	EtcdEndpoints          []string
	EtcdUser               string
	EtcdPass               string
	EtcdDialTimeout        time.Duration
	EtcdPrefix             string
	HeartbeatTTL           time.Duration
	LogHeartbeat           bool
	SyncServersInterval    time.Duration
	RevokeTimeout          time.Duration
	GrantLeaseTimeout      time.Duration
	GrantLeaseMaxRetries   int
	GrantLeaseInterval     time.Duration
	ShutdownDelay          time.Duration
	ServerTypesBlacklist   []string
	SyncServersParallelism int
}

// NewDefaultEtcdServiceDiscoveryConfig Etcd service discovery default config
func NewDefaultEtcdServiceDiscoveryConfig() EtcdServiceDiscoveryConfig {
	return EtcdServiceDiscoveryConfig{
		EtcdEndpoints:          []string{"localhost:2379"},
		EtcdUser:               "",
		EtcdPass:               "",
		EtcdDialTimeout:        time.Duration(5 * time.Second),
		EtcdPrefix:             "pitaya/",
		HeartbeatTTL:           time.Duration(60 * time.Second),
		LogHeartbeat:           false,
		SyncServersInterval:    time.Duration(120 * time.Second),
		RevokeTimeout:          time.Duration(5 * time.Second),
		GrantLeaseTimeout:      time.Duration(60 * time.Second),
		GrantLeaseMaxRetries:   15,
		GrantLeaseInterval:     time.Duration(5 * time.Second),
		ShutdownDelay:          time.Duration(10 * time.Millisecond),
		ServerTypesBlacklist:   nil,
		SyncServersParallelism: 10,
	}
}

// NewEtcdServiceDiscoveryConfig Etcd service discovery config with default config paths
func NewEtcdServiceDiscoveryConfig(config *Config) EtcdServiceDiscoveryConfig {
	return EtcdServiceDiscoveryConfig{
		EtcdEndpoints:          config.GetStringSlice("pitaya.cluster.sd.etcd.endpoints"),
		EtcdUser:               config.GetString("pitaya.cluster.sd.etcd.user"),
		EtcdPass:               config.GetString("pitaya.cluster.sd.etcd.pass"),
		EtcdDialTimeout:        config.GetDuration("pitaya.cluster.sd.etcd.dialtimeout"),
		EtcdPrefix:             config.GetString("pitaya.cluster.sd.etcd.prefix"),
		HeartbeatTTL:           config.GetDuration("pitaya.cluster.sd.etcd.heartbeat.ttl"),
		LogHeartbeat:           config.GetBool("pitaya.cluster.sd.etcd.heartbeat.log"),
		SyncServersInterval:    config.GetDuration("pitaya.cluster.sd.etcd.syncservers.interval"),
		RevokeTimeout:          config.GetDuration("pitaya.cluster.sd.etcd.revoke.timeout"),
		GrantLeaseTimeout:      config.GetDuration("pitaya.cluster.sd.etcd.grantlease.timeout"),
		GrantLeaseMaxRetries:   config.GetInt("pitaya.cluster.sd.etcd.grantlease.maxretries"),
		GrantLeaseInterval:     config.GetDuration("pitaya.cluster.sd.etcd.grantlease.retryinterval"),
		ShutdownDelay:          config.GetDuration("pitaya.cluster.sd.etcd.shutdown.delay"),
		ServerTypesBlacklist:   config.GetStringSlice("pitaya.cluster.sd.etcd.servertypeblacklist"),
		SyncServersParallelism: config.GetInt("pitaya.cluster.sd.etcd.syncserversparallelism"),
	}
}

// NewDefaultCustomMetricsSpec returns an empty *CustomMetricsSpec
func NewDefaultCustomMetricsSpec() models.CustomMetricsSpec {
	return models.CustomMetricsSpec{
		Summaries: []*models.Summary{},
		Gauges:    []*models.Gauge{},
		Counters:  []*models.Counter{},
	}
}

// NewCustomMetricsSpec returns a *CustomMetricsSpec by reading config key (DEPRECATED)
func NewCustomMetricsSpec(config *Config) models.CustomMetricsSpec {
	var spec models.CustomMetricsSpec

	err := config.UnmarshalKey("pitaya.metrics.custom", &spec)
	if err != nil {
		return NewDefaultCustomMetricsSpec()
	}

	return spec
}

// PrometheusConfig provides configuration for PrometheusReporter
type PrometheusConfig struct {
	Port             int
	Game             string
	AdditionalLabels map[string]string
	ConstLabels      map[string]string
}

// NewDefaultPrometheusConfig provides default configuration for PrometheusReporter
func NewDefaultPrometheusConfig() PrometheusConfig {
	return PrometheusConfig{
		Port:             9090,
		AdditionalLabels: map[string]string{},
		ConstLabels:      map[string]string{},
	}
}

// NewPrometheusConfig reads from config to build configuration for PrometheusReporter
func NewPrometheusConfig(config *Config) PrometheusConfig {
	return PrometheusConfig{
		Port:             config.GetInt("pitaya.metrics.prometheus.port"),
		Game:             config.GetString("pitaya.game"),
		AdditionalLabels: config.GetStringMapString("pitaya.metrics.additionalTags"),
		ConstLabels:      config.GetStringMapString("pitaya.metrics.constTags"),
	}
}

// StatsdConfig provides configuration for statsd
type StatsdConfig struct {
	Host        string
	Prefix      string
	Rate        float64
	ConstLabels map[string]string
}

// NewDefaultStatsdConfig provides default configuration for statsd
func NewDefaultStatsdConfig() StatsdConfig {
	return StatsdConfig{
		Host:        "localhost:9125",
		Prefix:      "pitaya.",
		Rate:        1,
		ConstLabels: map[string]string{},
	}
}

// NewStatsdConfig reads from config to build configuration for statsd
func NewStatsdConfig(config *Config) StatsdConfig {
	rate, err := strconv.ParseFloat(config.GetString("pitaya.metrics.statsd.rate"), 64)
	if err != nil {
		panic(err.Error())
	}

	statsdConfig := StatsdConfig{
		Host:        config.GetString("pitaya.metrics.statsd.host"),
		Prefix:      config.GetString("pitaya.metrics.statsd.prefix"),
		Rate:        rate,
		ConstLabels: config.GetStringMapString("pitaya.metrics.constTags"),
	}

	return statsdConfig
}

// WorkerConfig provides worker configuration
type WorkerConfig struct {
	ServerURL   string
	Pool        string
	Password    string
	Namespace   string
	Concurrency int
}

// NewDefaultWorkerConfig provides worker default configuration
func NewDefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		ServerURL:   "localhost:6379",
		Pool:        "10",
		Concurrency: 1,
	}
}

// NewWorkerConfig provides worker configuration based on default string paths
func NewWorkerConfig(config *Config) WorkerConfig {
	return WorkerConfig{
		ServerURL:   config.GetString("pitaya.worker.redis.url"),
		Pool:        config.GetString("pitaya.worker.redis.pool"),
		Password:    config.GetString("pitaya.worker.redis.password"),
		Namespace:   config.GetString("pitaya.worker.namespace"),
		Concurrency: config.GetInt("pitaya.worker.concurrency"),
	}
}

// EnqueueOpts has retry options for worker
type EnqueueOpts struct {
	RetryEnabled      bool
	MaxRetries        int
	ExponentialFactor int
	MinDelayToRetry   int
	MaxDelayToRetry   int
	MaxRandom         int
}

// NewDefaultEnqueueOpts provides default EnqueueOpts
func NewDefaultEnqueueOpts() EnqueueOpts {
	return EnqueueOpts{
		RetryEnabled:      true,
		MaxRetries:        2,
		ExponentialFactor: 5,
		MinDelayToRetry:   10,
		MaxDelayToRetry:   10,
		MaxRandom:         0,
	}
}

// NewEnqueueOpts reads from config to build *EnqueueOpts
func NewEnqueueOpts(config *Config) EnqueueOpts {
	return EnqueueOpts{
		RetryEnabled:      config.GetBool("pitaya.worker.retry.enabled"),
		MaxRetries:        config.GetInt("pitaya.worker.retry.max"),
		ExponentialFactor: config.GetInt("pitaya.worker.retry.exponential"),
		MinDelayToRetry:   config.GetInt("pitaya.worker.retry.minDelay"),
		MaxDelayToRetry:   config.GetInt("pitaya.worker.retry.maxDelay"),
		MaxRandom:         config.GetInt("pitaya.worker.retry.maxRandom"),
	}
}

// MemoryGroupConfig provides configuration for MemoryGroup
type MemoryGroupConfig struct {
	TickDuration time.Duration
}

// NewDefaultMemoryGroupConfig returns a new, default group instance
func NewDefaultMemoryGroupConfig() MemoryGroupConfig {
	return MemoryGroupConfig{TickDuration: time.Duration(30 * time.Second)}
}

// NewMemoryGroupConfig returns a new, default group instance
func NewMemoryGroupConfig(conf *Config) MemoryGroupConfig {
	return MemoryGroupConfig{TickDuration: conf.GetDuration("pitaya.groups.memory.tickduration")}
}

// EtcdGroupServiceConfig provides ETCD configuration
type EtcdGroupServiceConfig struct {
	DialTimeout        time.Duration
	Endpoints          []string
	Prefix             string
	TransactionTimeout time.Duration
}

// NewDefaultEtcdGroupServiceConfig provides default ETCD configuration
func NewDefaultEtcdGroupServiceConfig() EtcdGroupServiceConfig {
	return EtcdGroupServiceConfig{
		DialTimeout:        time.Duration(5 * time.Second),
		Endpoints:          []string{"localhost:2379"},
		Prefix:             "pitaya/",
		TransactionTimeout: time.Duration(5 * time.Second),
	}
}

// NewEtcdGroupServiceConfig reads from config to build ETCD configuration
func NewEtcdGroupServiceConfig(config *Config) EtcdGroupServiceConfig {
	return EtcdGroupServiceConfig{
		DialTimeout:        config.GetDuration("pitaya.groups.etcd.dialtimeout"),
		Endpoints:          config.GetStringSlice("pitaya.groups.etcd.endpoints"),
		Prefix:             config.GetString("pitaya.groups.etcd.prefix"),
		TransactionTimeout: config.GetDuration("pitaya.groups.etcd.transactiontimeout"),
	}
}

// ETCDBindingConfig provides configuration for ETCDBindingStorage
type ETCDBindingConfig struct {
	DialTimeout time.Duration
	Endpoints   []string
	Prefix      string
	LeaseTTL    time.Duration
}

// NewDefaultETCDBindingConfig provides default configuration for ETCDBindingStorage
func NewDefaultETCDBindingConfig() ETCDBindingConfig {
	return ETCDBindingConfig{
		DialTimeout: time.Duration(5 * time.Second),
		Endpoints:   []string{"localhost:2379"},
		Prefix:      "pitaya/",
		LeaseTTL:    time.Duration(5 * time.Hour),
	}
}

// NewETCDBindingConfig reads from config to build ETCDBindingStorage configuration
func NewETCDBindingConfig(config *Config) ETCDBindingConfig {
	return ETCDBindingConfig{
		DialTimeout: config.GetDuration("pitaya.modules.bindingstorage.etcd.dialtimeout"),
		Endpoints:   config.GetStringSlice("pitaya.modules.bindingstorage.etcd.endpoints"),
		Prefix:      config.GetString("pitaya.modules.bindingstorage.etcd.prefix"),
		LeaseTTL:    config.GetDuration("pitaya.modules.bindingstorage.etcd.leasettl"),
	}
}

// RateLimitingConfig rate limits config
type RateLimitingConfig struct {
	Limit        int
	Interval     time.Duration
	ForceDisable bool
}

// NewDefaultRateLimitingConfig rate limits default config
func NewDefaultRateLimitingConfig() RateLimitingConfig {
	return RateLimitingConfig{
		Limit:        20,
		Interval:     time.Duration(time.Second),
		ForceDisable: false,
	}
}

// NewRateLimitingConfig reads from config to build rate limiting configuration
func NewRateLimitingConfig(config *Config) RateLimitingConfig {
	return RateLimitingConfig{
		Limit:        config.GetInt("pitaya.conn.ratelimiting.limit"),
		Interval:     config.GetDuration("pitaya.conn.ratelimiting.interval"),
		ForceDisable: config.GetBool("pitaya.conn.ratelimiting.forcedisable"),
	}
}
