*************
Configuration
*************

Pitaya uses Viper to control its configuration. Below we describe the configuration variables split by topic. We judge the default values are good for most cases, but might need to be changed for some use cases.

Service Discovery
=================

These configuration values configure service discovery for the default etcd service discovery module.
 They only need to be set if the application runs in cluster mode.

.. list-table::
  :widths: 15 10 10 50
  :header-rows: 1
  :stub-columns: 1

  * - Configuration
    - Default value
    - Type
    - Description
  * - pitaya.cluster.sd.etcd.dialtimeout
    - 5s
    - time.Time
    - Dial timeout value passed to the service discovery etcd client
  * - pitaya.cluster.sd.etcd.endpoints
    - localhost:2379
    - string
    - List of comma separated etcd endpoints
  * - pitaya.cluster.sd.etcd.user
    - 
    - string
    - Username to connect to etcd
  * - pitaya.cluster.sd.etcd.pass
    - 
    - string
    - Password to connect to etcd
  * - pitaya.cluster.sd.etcd.heartbeat.ttl
    - 60s
    - time.Time
    - Hearbeat interval for the etcd lease
  * - pitaya.cluster.sd.etcd.grantlease.timeout
    - 60s
    - time.Duration
    - Timeout for etcd lease
  * - pitaya.cluster.sd.etcd.grantlease.maxretries
    - 15
    - int
    - Maximum number of attempts to etcd grant lease
  * - pitaya.cluster.sd.etcd.grantlease.retryinterval
    - 5s
    - time.Duration
    - Interval between each grant lease attempt
  * - pitaya.cluster.sd.etcd.revoke.timeout
    - 5s
    - time.Duration
    - Timeout for etcd's revoke function
  * - pitaya.cluster.sd.etcd.heartbeat.log
    - false
    - bool
    - Whether etcd heartbeats should be logged in debug mode
  * - pitaya.cluster.sd.etcd.prefix
    - pitaya/
    - string
    - Prefix used to avoid collisions with different pitaya applications, servers must have the same prefix to be able to see each other
  * - pitaya.cluster.sd.etcd.syncservers.interval
    - 120s
    - time.Duration
    - Interval between server syncs performed by the service discovery module
  * - pitaya.cluster.sd.etcd.shutdown.delay
    - 10ms
    - time.Duration
    - Time to wait to shutdown after deregistering from service discovery
  * - pitaya.cluster.sd.etcd.servertypeblacklist
    - nil
    - []string
    - A list of server types that should be ignored by the service discovery
  * - pitaya.cluster.sd.etcd.syncserversparallelism
    - 10
    - int
    - The number of goroutines that should be used while getting server information on etcd initialization

RPC Service
===========

The configurations only need to be set if the RPC Service is enabled with the given type.

.. list-table::
  :widths: 15 10 10 50
  :header-rows: 1
  :stub-columns: 1

  * - Configuration
    - Default value
    - Type
    - Description
  * - pitaya.buffer.cluster.rpc.server.nats.messages
    - 75
    - int
    - Size of the buffer that for the nats RPC server accepts before starting to drop incoming messages
  * - pitaya.buffer.cluster.rpc.server.nats.push
    - 100
    - int
    - Size of the buffer that the nats RPC server creates for push messages
  * - pitaya.cluster.rpc.client.grpc.dialtimeout
    - 5s
    - time.Time
    - Timeout for the gRPC client to establish the connection
  * - pitaya.cluster.rpc.client.grpc.lazyconnection
    - false
    - bool
    - Whether the gRPC client should use a lazy connection, that is, connect only when a request is made to that server
  * - pitaya.cluster.rpc.client.grpc.requesttimeout
    - 5s
    - time.Time
    - Request timeout for RPC calls with the gRPC client
  * - pitaya.cluster.rpc.client.nats.connect
    - nats://localhost:4222
    - string
    - Nats address for the client
  * - pitaya.cluster.rpc.client.nats.connectiontimeout
    - 5s
    - time.Duration
    - Timeout for the nats client to establish the connection
  * - pitaya.cluster.rpc.client.nats.requesttimeout
    - 5s
    - time.Time
    - Request timeout for RPC calls with the nats client
  * - pitaya.cluster.rpc.client.nats.maxreconnectionretries
    - 15
    - int
    - Maximum number of retries to reconnect to nats for the client
  * - pitaya.cluster.rpc.server.nats.connect
    - nats://localhost:4222
    - string
    - Nats address for the server
  * - pitaya.cluster.rpc.server.nats.connectiontimeout
    - 5s
    - time.Duration
    - Timeout for the nats server to establish the connection
  * - pitaya.cluster.rpc.server.nats.maxreconnectionretries
    - 15
    - int
    - Maximum number of retries to reconnect to nats for the server
  * - pitaya.cluster.rpc.server.grpc.port
    - 3434
    - int
    - The port that the gRPC server listens to
  * - pitaya.concurrency.remote.service
    - 30
    - int
    - Number of goroutines processing messages at the remote service for the nats RPC service
  * - pitaya.worker.redis.url
    - localhost:6379
    - string
    - Redis url pitaya workers use to register jobs
  * - pitaya.worker.redis.pool
    - 10
    - string
    - Number of connections to keep with Redis
  * - pitaya.worker.redis.password
    - ""
    - string
    - Redis password to connect to pitaya workers redis
  * - pitaya.worker.concurrency
    - 1
    - int
    - Number of workers to execute job
  * - pitaya.worker.namespace
    - ""
    - string
    - Worker namespace, can be used to differ stacks in a blue-green deployment
  * - pitaya.worker.retry.enabled
    - true
    - bool
    - If true, retry job if errored for max times
  * - pitaya.worker.retry.max
    - 5
    - int
    - Max number of job retries
  * - pitaya.worker.retry.exponential
    - 2
    - int
    - Retry job after backoff of nRetry**2
  * - pitaya.worker.retry.minDelay
    - 0
    - int
    - Min time to wait on backoff to retry job
  * - pitaya.worker.retry.maxDelay
    - 10
    - int
    - Max time to wait on backoff to retry job
  * - pitaya.worker.retry.maxRandom
    - 10
    - int
    - Random time to wait during backoff

Connection
==========

.. list-table::
  :widths: 15 10 10 50
  :header-rows: 1
  :stub-columns: 1

  * - Configuration
    - Default value
    - Type
    - Description
  * - pitaya.handler.messages.compression
    - true
    - bool
    - Whether messages between client and server should be compressed
  * - pitaya.heartbeat.interval
    - 30s
    - time.Time
    - Keepalive heartbeat interval for the client connection
  * - pitaya.conn.ratelimiting.interval
    - 1s
    - time.Duration
    - Window of time to count requests
  * - pitaya.conn.ratelimiting.limit
    - 20
    - int
    - Max number of requests allowed in a interval
  * - pitaya.conn.ratelimiting.forcedisable
    - false
    - bool
    - If true, ignores rate limiting even when added with WithWrappers

Metrics Reporting
=================

.. list-table::
  :widths: 15 10 10 50
  :header-rows: 1
  :stub-columns: 1

  * - Configuration
    - Default value
    - Type
    - Description
  * - pitaya.metrics.statsd.enabled
    - false
    - bool
    - Whether statsd reporting should be enabled
  * - pitaya.metrics.statsd.host
    - localhost:9125
    - string
    - Address of the statsd server to send the metrics to
  * - pitaya.metrics.statsd.prefix
    - pitaya.
    - string
    - Prefix of the metrics reported to statsd
  * - pitaya.metrics.statsd.rate
    - 1
    - int
    - Statsd metrics rate
  * - pitaya.metrics.prometheus.enabled
    - false
    - bool
    - Whether prometheus reporting should be enabled
  * - pitaya.metrics.prometheus.port
    - 9090
    - int
    - Port to expose prometheus metrics
  * - pitaya.metrics.constTags
    - map[string]string{}
    - map[string]string
    - Constant tags to be added to reported metrics
  * - pitaya.metrics.additionalTags
    - map[string]string{}
    - map[string]string
    - Additional tags to reported metrics, the map is from tag to default value
  * - pitaya.metrics.periodicMetrics.period
    - 15s
    - string
    - Period that system metrics will be reported
  * - pitaya.metrics.custom.counters
    - []map[string]interface{}
    - []map[string]interface
    - Custom metrics counter
  * - pitaya.metrics.custom.counters[].Subsystem
    - ""
    - string
    - Custom counter subsystem name
  * - pitaya.metrics.custom.counters[].Name
    - ""
    - string
    - Custom counter name, must not be empty
  * - pitaya.metrics.custom.counters[].Help
    - ""
    - string
    - Custom counter help which explain what is the metric, must not be empty
  * - pitaya.metrics.custom.counters[].Labels
    - []string{}
    - []string
    - Custom counter labels the metric will carry
  * - pitaya.metrics.custom.gauges
    - []map[string]interface{}
    - []map[string]interface
    - Custom metrics gauge 
  * - pitaya.metrics.custom.gauges[].Subsystem
    - ""
    - string
    - Custom gauge subsystem name
  * - pitaya.metrics.custom.gauges[].Name
    - ""
    - string
    - Custom gauge name, must not be empty
  * - pitaya.metrics.custom.gauges[].Help
    - ""
    - string
    - Custom gauge help which explain what is the metric, must not be empty
  * - pitaya.metrics.custom.gauges[].Labels
    - []string{}
    - []string
    - Custom gauge labels the metric will carry
  * - pitaya.metrics.custom.summaries
    - []map[string]interface{}
    - []map[string]interface
    - Custom metrics summary 
  * - pitaya.metrics.custom.summaries[].Subsystem
    - ""
    - string
    - Custom summary subsystem name
  * - pitaya.metrics.custom.summaries[].Name
    - ""
    - string
    - Custom summary name, must not be empty
  * - pitaya.metrics.custom.summaries[].Help
    - ""
    - string
    - Custom summary help which explain what is the metric, must not be empty
  * - pitaya.metrics.custom.summaries[].Labels
    - []string{}
    - []string
    - Custom summary labels the metric will carry
  * - pitaya.metrics.custom.summaries[].Objectives
    - map[float64]float64
    - map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
    - Custom summary objectives with quantiles 

Concurrency
===========

.. list-table::
  :widths: 15 10 10 50
  :header-rows: 1
  :stub-columns: 1

  * - Configuration
    - Default value
    - Type
    - Description
  * - pitaya.buffer.agent.messages
    - 100
    - int
    - Buffer size for received client messages for each agent
  * - pitaya.buffer.handler.localprocess
    - 20
    - int
    - Buffer size for messages received by the handler and processed locally
  * - pitaya.buffer.handler.remoteprocess
    - 20
    - int
    - Buffer size for messages received by the handler and forwarded to remote servers
  * - pitaya.concurrency.handler.dispatch
    - 25
    - int
    - Number of goroutines processing messages at the handler service

Modules
=======

These configurations are only used if the modules are created. It is recommended to use Binding Storage module with gRPC RPC service to be able to use all RPC service features.

.. list-table::
  :widths: 15 10 10 50
  :header-rows: 1
  :stub-columns: 1

  * - Configuration
    - Default value
    - Type
    - Description
  * - pitaya.session.unique
    - true
    - bool
    - Whether Pitaya should enforce unique sessions for the clients, enabling the unique sessions module
  * - pitaya.modules.bindingstorage.etcd.endpoints
    - localhost:2379
    - string
    - Comma separated list of etcd endpoints to be used by the binding storage module, should be the same as the service discovery etcd
  * - pitaya.modules.bindingstorage.etcd.prefix
    - pitaya/
    - string
    - Prefix used for etcd, should be the same as the service discovery
  * - pitaya.modules.bindingstorage.etcd.dialtimeout
    - 5s
    - time.Time
    - Timeout to establish the etcd connection
  * - pitaya.modules.bindingstorage.etcd.leasettl
    - 1h
    - time.Time
    - Duration of the etcd lease before automatic renewal

Default Pipelines
=================

These configurations control if the default pipelines should be enabled or not

.. list-table::
  :widths: 15 10 10 50
  :header-rows: 1
  :stub-columns: 1

  * - Configuration
    - Default value
    - Type
    - Description
  * - pitaya.defaultpipelines.structvalidation.enabled
    - false
    - bool
    - Whether Pitaya should enable the default struct validator for handler arguments

Groups
=================

These configurations are used for group services implementations.

.. list-table::
  :widths: 15 10 10 50
  :header-rows: 1
  :stub-columns: 1

  * - Configuration
    - Default value
    - Type
    - Description
  * - pitaya.groups.etcd.endpoints
    - localhost:2379
    - string
    - Comma separated list of etcd endpoints to be used by the groups etcd service 
  * - pitaya.groups.etcd.prefix
    - pitaya/
    - string 
    - Prefix used for every group key in etcd
  * - pitaya.groups.etcd.dialtimeout
    - 5s
    - time.Time
    - Timeout to establish the etcd group connection
  * - pitaya.groups.etcd.transactiontimeout
    - 5s
    - time.Duration
    - Timeout to finish group request to Etcd
  * - pitaya.groups.memory.tickduration
    - 30s
    - time.Duration
    - Duration time of tick that will check if should delete group or not
