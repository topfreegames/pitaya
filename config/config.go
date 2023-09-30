package config

import (
	"time"

	"github.com/topfreegames/pitaya/v2/metrics/models"
)

// PitayaConfig provides configuration for a pitaya app
type PitayaConfig struct {
	Heartbeat struct {
		Interval time.Duration `mapstructure:"interval"`
	} `mapstructure:"heartbeat"`
	Handler struct {
		Messages struct {
			Compression bool `mapstructure:"compression"`
		} `mapstructure:"messages"`
	} `mapstructure:"handler"`
	Buffer struct {
		Agent struct {
			Messages int `mapstructure:"messages"`
		} `mapstructure:"agent"`
		Handler struct {
			LocalProcess  int `mapstructure:"localprocess"`
			RemoteProcess int `mapstructure:"remoteprocess"`
		} `mapstructure:"handler"`
	} `mapstructure:"buffer"`
	Concurrency struct {
		Handler struct {
			Dispatch int `mapstructure:"dispatch"`
		} `mapstructure:"handler"`
	} `mapstructure:"concurrency"`
	Session struct {
		Unique bool `mapstructure:"unique"`
		Drain  struct {
			Enabled bool          `mapstructure:"enabled"`
			Timeout time.Duration `mapstructure:"timeout"`
			Period  time.Duration `mapstructure:"period"`
		} `mapstructure:"drain"`
	} `mapstructure:"session"`
	Metrics struct {
		Period time.Duration `mapstructure:"period"`
	} `mapstructure:"metrics"`
	Acceptor struct {
		ProxyProtocol bool `mapstructure:"proxyprotocol"`
	} `mapstructure:"acceptor"`
}

// NewDefaultPitayaConfig provides default configuration for Pitaya App
func NewDefaultPitayaConfig() *PitayaConfig {
	return &PitayaConfig{
		Heartbeat: struct {
			Interval time.Duration `mapstructure:"interval"`
		}{
			Interval: time.Duration(30 * time.Second),
		},
		Handler: struct {
			Messages struct {
				Compression bool `mapstructure:"compression"`
			} `mapstructure:"messages"`
		}{
			Messages: struct {
				Compression bool `mapstructure:"compression"`
			}{
				Compression: true,
			},
		},
		Buffer: struct {
			Agent struct {
				Messages int `mapstructure:"messages"`
			} `mapstructure:"agent"`
			Handler struct {
				LocalProcess  int `mapstructure:"localprocess"`
				RemoteProcess int `mapstructure:"remoteprocess"`
			} `mapstructure:"handler"`
		}{
			Agent: struct {
				Messages int `mapstructure:"messages"`
			}{
				Messages: 100,
			},
			Handler: struct {
				LocalProcess  int `mapstructure:"localprocess"`
				RemoteProcess int `mapstructure:"remoteprocess"`
			}{
				LocalProcess:  20,
				RemoteProcess: 20,
			},
		},
		Concurrency: struct {
			Handler struct {
				Dispatch int `mapstructure:"dispatch"`
			} `mapstructure:"handler"`
		}{
			Handler: struct {
				Dispatch int `mapstructure:"dispatch"`
			}{
				Dispatch: 25,
			},
		},
		Session: struct {
			Unique bool `mapstructure:"unique"`
			Drain  struct {
				Enabled bool          `mapstructure:"enabled"`
				Timeout time.Duration `mapstructure:"timeout"`
				Period  time.Duration `mapstructure:"period"`
			} `mapstructure:"drain"`
		}{
			Unique: true,
			Drain: struct {
				Enabled bool          `mapstructure:"enabled"`
				Timeout time.Duration `mapstructure:"timeout"`
				Period  time.Duration `mapstructure:"period"`
			}{
				Enabled: false,
				Timeout: time.Duration(6 * time.Hour),
				Period:  time.Duration(5 * time.Second),
			},
		},
		Metrics: struct {
			Period time.Duration `mapstructure:"period"`
		}{
			Period: time.Duration(15 * time.Second),
		},
		Acceptor: struct {
			ProxyProtocol bool `mapstructure:"proxyprotocol"`
		}{
			ProxyProtocol: false,
		},
	}
}

// NewPitayaConfig returns a config instance with values extracted from default config paths
func NewPitayaConfig(config *Config) *PitayaConfig {
	conf := NewDefaultPitayaConfig()
	if err := config.UnmarshalKey("pitaya", &conf); err != nil {
		panic(err)
	}
	return conf
}

// BuilderConfig provides configuration for Builder
type BuilderConfig struct {
	Pitaya  PitayaConfig
	Metrics struct {
		Prometheus struct {
			Enabled bool `mapstructure:"enabled"`
		} `mapstructure:"prometheus"`
		Statsd struct {
			Enabled bool `mapstructure:"enabled"`
		} `mapstructure:"statsd"`
	} `mapstructure:"metrics"`
	DefaultPipelines struct {
		StructValidation struct {
			Enabled bool `mapstructure:"enabled"`
		} `mapstructure:"structvalidation"`
	} `mapstructure:"defaultpipelines"`
}

// NewDefaultBuilderConfig provides default builder configuration
func NewDefaultBuilderConfig() *BuilderConfig {
	return &BuilderConfig{
		Pitaya: *NewDefaultPitayaConfig(),
		Metrics: struct {
			Prometheus struct {
				Enabled bool `mapstructure:"enabled"`
			} `mapstructure:"prometheus"`
			Statsd struct {
				Enabled bool `mapstructure:"enabled"`
			} `mapstructure:"statsd"`
		}{
			Prometheus: struct {
				Enabled bool `mapstructure:"enabled"`
			}{
				Enabled: false,
			},
			Statsd: struct {
				Enabled bool `mapstructure:"enabled"`
			}{
				Enabled: false,
			},
		},
		DefaultPipelines: struct {
			StructValidation struct {
				Enabled bool `mapstructure:"enabled"`
			} `mapstructure:"structvalidation"`
		}{
			StructValidation: struct {
				Enabled bool `mapstructure:"enabled"`
			}{
				Enabled: false,
			},
		},
	}
}

// NewBuilderConfig reads from config to build builder configuration
func NewBuilderConfig(config *Config) *BuilderConfig {
	conf := NewDefaultBuilderConfig()
	if err := config.Unmarshal(&conf); err != nil {
		panic(err)
	}
	if err := config.UnmarshalKey("pitaya", &conf); err != nil {
		panic(err)
	}
	return conf
}

// GRPCClientConfig rpc client config struct
type GRPCClientConfig struct {
	DialTimeout    time.Duration `mapstructure:"dialtimeout"`
	LazyConnection bool          `mapstructure:"lazyconnection"`
	RequestTimeout time.Duration `mapstructure:"requesttimeout"`
}

// NewDefaultGRPCClientConfig rpc client default config struct
func NewDefaultGRPCClientConfig() *GRPCClientConfig {
	return &GRPCClientConfig{
		DialTimeout:    time.Duration(5 * time.Second),
		LazyConnection: false,
		RequestTimeout: time.Duration(5 * time.Second),
	}
}

// NewGRPCClientConfig reads from config to build GRPCCLientConfig
func NewGRPCClientConfig(config *Config) *GRPCClientConfig {
	conf := NewDefaultGRPCClientConfig()
	if err := config.UnmarshalKey("pitaya.cluster.rpc.client.grpc", &conf); err != nil {
		panic(err)
	}
	return conf
}

// GRPCServerConfig provides configuration for GRPCServer
type GRPCServerConfig struct {
	Port int `mapstructure:"port"`
}

// NewDefaultGRPCServerConfig returns a default GRPCServerConfig
func NewDefaultGRPCServerConfig() *GRPCServerConfig {
	return &GRPCServerConfig{
		Port: 3434,
	}
}

// NewGRPCServerConfig reads from config to build GRPCServerConfig
func NewGRPCServerConfig(config *Config) *GRPCServerConfig {
	return &GRPCServerConfig{
		Port: config.GetInt("pitaya.cluster.rpc.server.grpc.port"),
	}
}

// NatsRPCClientConfig provides nats client configuration
type NatsRPCClientConfig struct {
	Connect                string        `mapstructure:"connect"`
	MaxReconnectionRetries int           `mapstructure:"maxreconnectionretries"`
	RequestTimeout         time.Duration `mapstructure:"requesttimeout"`
	ConnectionTimeout      time.Duration `mapstructure:"connectiontimeout"`
}

// NewDefaultNatsRPCClientConfig provides default nats client configuration
func NewDefaultNatsRPCClientConfig() *NatsRPCClientConfig {
	return &NatsRPCClientConfig{
		Connect:                "nats://localhost:4222",
		MaxReconnectionRetries: 15,
		RequestTimeout:         time.Duration(5 * time.Second),
		ConnectionTimeout:      time.Duration(2 * time.Second),
	}
}

// NewNatsRPCClientConfig reads from config to build nats client configuration
func NewNatsRPCClientConfig(config *Config) *NatsRPCClientConfig {
	conf := NewDefaultNatsRPCClientConfig()
	if err := config.UnmarshalKey("pitaya.cluster.rpc.client.nats", &conf); err != nil {
		panic(err)
	}
	return conf
}

// NatsRPCServerConfig provides nats server configuration
type NatsRPCServerConfig struct {
	Connect                string `mapstructure:"connect"`
	MaxReconnectionRetries int    `mapstructure:"maxreconnectionretries"`
	Buffer                 struct {
		Messages int `mapstructure:"messages"`
		Push     int `mapstructure:"push"`
	} `mapstructure:"buffer"`
	Services          int           `mapstructure:"services"`
	ConnectionTimeout time.Duration `mapstructure:"connectiontimeout"`
}

// NewDefaultNatsRPCServerConfig provides default nats server configuration
func NewDefaultNatsRPCServerConfig() *NatsRPCServerConfig {
	return &NatsRPCServerConfig{
		Connect:                "nats://localhost:4222",
		MaxReconnectionRetries: 15,
		Buffer: struct {
			Messages int `mapstructure:"messages"`
			Push     int `mapstructure:"push"`
		}{
			Messages: 75,
			Push:     100,
		},
		Services:          30,
		ConnectionTimeout: time.Duration(2 * time.Second),
	}
}

// NewNatsRPCServerConfig reads from config to build nats server configuration
func NewNatsRPCServerConfig(config *Config) *NatsRPCServerConfig {
	conf := NewDefaultNatsRPCServerConfig()
	if err := config.UnmarshalKey("pitaya.cluster.rpc.server.nats", &conf); err != nil {
		panic(err)
	}
	return conf
}

// InfoRetrieverConfig provides InfoRetriever configuration
type InfoRetrieverConfig struct {
	Region string `mapstructure:"region"`
}

// NewDefaultInfoRetrieverConfig provides default configuration for InfoRetriever
func NewDefaultInfoRetrieverConfig() *InfoRetrieverConfig {
	return &InfoRetrieverConfig{
		Region: "",
	}
}

// NewInfoRetrieverConfig reads from config to build configuration for InfoRetriever
func NewInfoRetrieverConfig(c *Config) *InfoRetrieverConfig {
	conf := NewDefaultInfoRetrieverConfig()
	if err := c.UnmarshalKey("pitaya.cluster.info", &conf); err != nil {
		panic(err)
	}
	return conf
}

// EtcdServiceDiscoveryConfig Etcd service discovery config
type EtcdServiceDiscoveryConfig struct {
	Endpoints   []string      `mapstructure:"endpoints"`
	User        string        `mapstructure:"user"`
	Pass        string        `mapstructure:"pass"`
	DialTimeout time.Duration `mapstructure:"dialtimeout"`
	Prefix      string        `mapstructure:"prefix"`
	Heartbeat   struct {
		TTL time.Duration `mapstructure:"ttl"`
		Log bool          `mapstructure:"log"`
	} `mapstructure:"heartbeat"`
	SyncServers struct {
		Interval    time.Duration `mapstructure:"interval"`
		Parallelism int           `mapstructure:"parallelism"`
	} `mapstructure:"syncservers"`
	Revoke struct {
		Timeout time.Duration `mapstructure:"timeout"`
	} `mapstructure:"revoke"`
	GrantLease struct {
		Timeout       time.Duration `mapstructure:"timeout"`
		MaxRetries    int           `mapstructure:"maxretries"`
		RetryInterval time.Duration `mapstructure:"retryinterval"`
	} `mapstructure:"grantlease"`
	Shutdown struct {
		Delay time.Duration `mapstructure:"delay"`
	} `mapstructure:"shutdown"`
	ServerTypesBlacklist []string `mapstructure:"servertypesblacklist"`
}

// NewDefaultEtcdServiceDiscoveryConfig Etcd service discovery default config
func NewDefaultEtcdServiceDiscoveryConfig() *EtcdServiceDiscoveryConfig {
	return &EtcdServiceDiscoveryConfig{
		Endpoints:   []string{"localhost:2379"},
		User:        "",
		Pass:        "",
		DialTimeout: time.Duration(5 * time.Second),
		Prefix:      "pitaya/",
		Heartbeat: struct {
			TTL time.Duration `mapstructure:"ttl"`
			Log bool          `mapstructure:"log"`
		}{
			TTL: time.Duration(60 * time.Second),
			Log: false,
		},
		SyncServers: struct {
			Interval    time.Duration `mapstructure:"interval"`
			Parallelism int           `mapstructure:"parallelism"`
		}{
			Interval:    time.Duration(120 * time.Second),
			Parallelism: 10,
		},
		Revoke: struct {
			Timeout time.Duration `mapstructure:"timeout"`
		}{
			Timeout: time.Duration(5 * time.Second),
		},
		GrantLease: struct {
			Timeout       time.Duration `mapstructure:"timeout"`
			MaxRetries    int           `mapstructure:"maxretries"`
			RetryInterval time.Duration `mapstructure:"retryinterval"`
		}{
			Timeout:       time.Duration(60 * time.Second),
			MaxRetries:    15,
			RetryInterval: time.Duration(5 * time.Second),
		},
		Shutdown: struct {
			Delay time.Duration `mapstructure:"delay"`
		}{
			Delay: time.Duration(300 * time.Millisecond),
		},
		ServerTypesBlacklist: nil,
	}
}

// NewEtcdServiceDiscoveryConfig Etcd service discovery config with default config paths
func NewEtcdServiceDiscoveryConfig(config *Config) *EtcdServiceDiscoveryConfig {
	conf := NewDefaultEtcdServiceDiscoveryConfig()
	if err := config.UnmarshalKey("pitaya.cluster.sd.etcd", &conf); err != nil {
		panic(err)
	}
	return conf
}

// NewDefaultCustomMetricsSpec returns an empty *CustomMetricsSpec
func NewDefaultCustomMetricsSpec() *models.CustomMetricsSpec {
	return &models.CustomMetricsSpec{
		Summaries: []*models.Summary{},
		Gauges:    []*models.Gauge{},
		Counters:  []*models.Counter{},
	}
}

// NewCustomMetricsSpec returns a *CustomMetricsSpec by reading config key (DEPRECATED)
func NewCustomMetricsSpec(config *Config) *models.CustomMetricsSpec {
	spec := &models.CustomMetricsSpec{}

	if err := config.UnmarshalKey("pitaya.metrics.custom", &spec); err != nil {
		return NewDefaultCustomMetricsSpec()
	}

	return spec
}

// PrometheusConfig provides configuration for PrometheusReporter
type PrometheusConfig struct {
	Prometheus struct {
		Port             int               `mapstructure:"port"`
		AdditionalLabels map[string]string `mapstructure:"additionallabels"`
	} `mapstructure:"prometheus"`
	Game        string            `mapstructure:"game"`
	ConstLabels map[string]string `mapstructure:"constlabels"`
}

// NewDefaultPrometheusConfig provides default configuration for PrometheusReporter
func NewDefaultPrometheusConfig() *PrometheusConfig {
	return &PrometheusConfig{
		Prometheus: struct {
			Port             int               `mapstructure:"port"`
			AdditionalLabels map[string]string `mapstructure:"additionallabels"`
		}{
			Port:             9090,
			AdditionalLabels: map[string]string{},
		},
		ConstLabels: map[string]string{},
	}
}

// NewPrometheusConfig reads from config to build configuration for PrometheusReporter
func NewPrometheusConfig(config *Config) *PrometheusConfig {
	conf := NewDefaultPrometheusConfig()
	if err := config.UnmarshalKey("pitaya.metrics", &conf); err != nil {
		panic(err)
	}
	return conf
}

// StatsdConfig provides configuration for statsd
type StatsdConfig struct {
	Statsd struct {
		Host   string  `mapstructure:"host"`
		Prefix string  `mapstructure:"prefix"`
		Rate   float64 `mapstructure:"rate"`
	} `mapstructure:"statsd"`
	ConstLabels map[string]string `mapstructure:"consttags"`
}

// NewDefaultStatsdConfig provides default configuration for statsd
func NewDefaultStatsdConfig() *StatsdConfig {
	return &StatsdConfig{
		Statsd: struct {
			Host   string  `mapstructure:"host"`
			Prefix string  `mapstructure:"prefix"`
			Rate   float64 `mapstructure:"rate"`
		}{
			Host:   "localhost:9125",
			Prefix: "pitaya.",
			Rate:   1,
		},
		ConstLabels: map[string]string{},
	}
}

// NewStatsdConfig reads from config to build configuration for statsd
func NewStatsdConfig(config *Config) *StatsdConfig {
	conf := NewDefaultStatsdConfig()
	if err := config.UnmarshalKey("pitaya.metrics", &conf); err != nil {
		panic(err)
	}
	return conf
}

// WorkerConfig provides worker configuration
type WorkerConfig struct {
	Redis struct {
		ServerURL string `mapstructure:"serverurl"`
		Pool      string `mapstructure:"pool"`
		Password  string `mapstructure:"password"`
	} `mapstructure:"redis"`
	Namespace   string `mapstructure:"namespace"`
	Concurrency int    `mapstructure:"concurrency"`
}

// NewDefaultWorkerConfig provides worker default configuration
func NewDefaultWorkerConfig() *WorkerConfig {
	return &WorkerConfig{
		Redis: struct {
			ServerURL string `mapstructure:"serverurl"`
			Pool      string `mapstructure:"pool"`
			Password  string `mapstructure:"password"`
		}{
			ServerURL: "localhost:6379",
			Pool:      "10",
		},
		Concurrency: 1,
	}
}

// NewWorkerConfig provides worker configuration based on default string paths
func NewWorkerConfig(config *Config) *WorkerConfig {
	conf := NewDefaultWorkerConfig()
	if err := config.UnmarshalKey("pitaya.worker", &conf); err != nil {
		panic(err)
	}
	return conf
}

// EnqueueOpts has retry options for worker
type EnqueueOpts struct {
	Enabled     bool `mapstructure:"enabled"`
	Max         int  `mapstructure:"max"`
	Exponential int  `mapstructure:"exponential"`
	MinDelay    int  `mapstructure:"mindelay"`
	MaxDelay    int  `mapstructure:"maxdelay"`
	MaxRandom   int  `mapstructure:"maxrandom"`
}

// NewDefaultEnqueueOpts provides default EnqueueOpts
func NewDefaultEnqueueOpts() *EnqueueOpts {
	return &EnqueueOpts{
		Enabled:     true,
		Max:         2,
		Exponential: 5,
		MinDelay:    10,
		MaxDelay:    10,
		MaxRandom:   0,
	}
}

// NewEnqueueOpts reads from config to build *EnqueueOpts
func NewEnqueueOpts(config *Config) *EnqueueOpts {
	conf := NewDefaultEnqueueOpts()
	if err := config.UnmarshalKey("pitaya.worker.retry", &conf); err != nil {
		panic(err)
	}
	return conf
}

// MemoryGroupConfig provides configuration for MemoryGroup
type MemoryGroupConfig struct {
	TickDuration time.Duration `mapstructure:"tickduration"`
}

// NewDefaultMemoryGroupConfig returns a new, default group instance
func NewDefaultMemoryGroupConfig() *MemoryGroupConfig {
	return &MemoryGroupConfig{TickDuration: time.Duration(30 * time.Second)}
}

// NewMemoryGroupConfig returns a new, default group instance
func NewMemoryGroupConfig(conf *Config) *MemoryGroupConfig {
	c := NewDefaultMemoryGroupConfig()
	if err := conf.UnmarshalKey("pitaya.groups.memory", &c); err != nil {
		panic(err)
	}
	return c
}

// EtcdGroupServiceConfig provides ETCD configuration
type EtcdGroupServiceConfig struct {
	DialTimeout        time.Duration `mapstructure:"dialtimeout"`
	Endpoints          []string      `mapstructure:"endpoints"`
	Prefix             string        `mapstructure:"prefix"`
	TransactionTimeout time.Duration `mapstructure:"transactiontimeout"`
}

// NewDefaultEtcdGroupServiceConfig provides default ETCD configuration
func NewDefaultEtcdGroupServiceConfig() *EtcdGroupServiceConfig {
	return &EtcdGroupServiceConfig{
		DialTimeout:        time.Duration(5 * time.Second),
		Endpoints:          []string{"localhost:2379"},
		Prefix:             "pitaya/",
		TransactionTimeout: time.Duration(5 * time.Second),
	}
}

// NewEtcdGroupServiceConfig reads from config to build ETCD configuration
func NewEtcdGroupServiceConfig(config *Config) *EtcdGroupServiceConfig {
	conf := NewDefaultEtcdGroupServiceConfig()
	if err := config.UnmarshalKey("pitaya.groups.etcd", &conf); err != nil {
		panic(err)
	}
	return conf
}

// ETCDBindingConfig provides configuration for ETCDBindingStorage
type ETCDBindingConfig struct {
	DialTimeout time.Duration `mapstructure:"dialtimeout"`
	Endpoints   []string      `mapstructure:"endpoints"`
	Prefix      string        `mapstructure:"prefix"`
	LeaseTTL    time.Duration `mapstructure:"leasettl"`
}

// NewDefaultETCDBindingConfig provides default configuration for ETCDBindingStorage
func NewDefaultETCDBindingConfig() *ETCDBindingConfig {
	return &ETCDBindingConfig{
		DialTimeout: time.Duration(5 * time.Second),
		Endpoints:   []string{"localhost:2379"},
		Prefix:      "pitaya/",
		LeaseTTL:    time.Duration(5 * time.Hour),
	}
}

// NewETCDBindingConfig reads from config to build ETCDBindingStorage configuration
func NewETCDBindingConfig(config *Config) *ETCDBindingConfig {
	conf := NewDefaultETCDBindingConfig()
	if err := config.UnmarshalKey("pitaya.modules.bindingstorage.etcd", &conf); err != nil {
		panic(err)
	}
	return conf
}

// RateLimitingConfig rate limits config
type RateLimitingConfig struct {
	Limit        int           `mapstructure:"limit"`
	Interval     time.Duration `mapstructure:"interval"`
	ForceDisable bool          `mapstructure:"forcedisable"`
}

// NewDefaultRateLimitingConfig rate limits default config
func NewDefaultRateLimitingConfig() *RateLimitingConfig {
	return &RateLimitingConfig{
		Limit:        20,
		Interval:     time.Duration(time.Second),
		ForceDisable: false,
	}
}

// NewRateLimitingConfig reads from config to build rate limiting configuration
func NewRateLimitingConfig(config *Config) *RateLimitingConfig {
	conf := NewDefaultRateLimitingConfig()
	if err := config.UnmarshalKey("pitaya.conn.ratelimiting", &conf); err != nil {
		panic(err)
	}
	return conf
}
