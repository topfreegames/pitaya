Configuration
=============

Pitaya uses Viper to control its configuration. Below we describe the configuration variables split by topic. We judge the default values are good for most cases, but might need to be changed for some use cases.

## Service Discovery

These configuration values configure service discovery for the default etcd service discovery module.
 They only need to be set if the application runs in cluster mode.

| Configuration | Default value | Type | Description |
|------------------------------------|---------------|-----------|------------------------------------------|
| pitaya.cluster.sd.etcd.dialtimeout | 5s | time.Time | Dial timeout value passed to the service discovery etcd client |
| pitaya.cluster.sd.etcd.endpoints | localhost:2379 | string | List of comma separated etcd endpoints |
| pitaya.cluster.sd.etcd.heartbeat.ttl | 60s | time.Time | Hearbeat interval for the etcd lease |
| pitaya.cluster.sd.etcd.heartbeat.log | false | bool | Whether etcd heartbeats should be logged in debug mode |
| pitaya.cluster.sd.etcd.prefix | pitaya/ | string | Prefix used to avoid collisions with different pitaya applications, servers must have the same prefix to be able to see each other |
| pitaya.cluster.sd.etcd.syncservers.interval | 120s | time.Time | Interval between server syncs performed by the service discovery module |

## RPC Service

The configurations only need to be set if the RPC Service is enabled with the given type.

| Configuration | Default value | Type | Description |
|------------------------------------|---------------|-----------|------------------------------------------|
| pitaya.buffer.cluster.rpc.server.nats.messages | 75 | int | Size of the buffer that for the nats RPC server accepts before starting to drop incoming messages |
| pitaya.buffer.cluster.rpc.server.nats.push | 100 | int | Size of the buffer that the nats RPC server creates for push messages |
| pitaya.cluster.rpc.client.grpc.requesttimeout | 5s | time.Time | Request timeout for RPC calls with the gRPC client |
| pitaya.cluster.rpc.client.grpc.dialtimeout | 5s | time.Time| Timeout for the gRPC client to establish the connection |
| pitaya.cluster.rpc.client.nats.connect | nats://localhost:4222 | string | Nats address for the client |
| pitaya.cluster.rpc.client.nats.requesttimeout | 5s | time.Time | Request timeout for RPC calls with the nats client |
| pitaya.cluster.rpc.client.nats.maxreconnectionretries | 15 | int | Maximum number of retries to reconnect to nats for the client |
| pitaya.cluster.rpc.server.nats.connect | nats://localhost:4222 | string | Nats address for the server |
| pitaya.cluster.rpc.server.nats.maxreconnectionretries | 15 | int | Maximum number of retries to reconnect to nats for the server |
| pitaya.cluster.rpc.server.grpc.port | 3434 | int | The port that the gRPC server listens to |
| pitaya.concurrency.remote.service | 30 | int | Number of goroutines processing messages at the remote service for the nats RPC service |

## Connection

| Configuration | Default value | Type | Description |
|------------------------------------|---------------|-----------|------------------------------------------|
| pitaya.handler.messages.compression | true | bool | Whether messages between client and server should be compressed |
| pitaya.heartbeat.interval | 30s | time.Time | Keepalive heartbeat interval for the client connection |

## Metrics Reporting

| Configuration | Default value | Type | Description |
|------------------------------------|---------------|-----------|------------------------------------------|
| pitaya.metrics.statsd.enabled | false | bool | Whether statsd reporting should be enabled |
| pitaya.metrics.statsd.host | localhost:9125 | string | Address of the statsd server to send the metrics to |
| pitaya.metrics.statsd.prefix | pitaya. | string | Prefix of the metrics reported to statsd |
| pitaya.metrics.statsd.rate | 1 | int | Statsd metrics rate |
| pitaya.metrics.prometheus.enabled | false | bool | Whether prometheus reporting should be enabled |
| pitaya.metrics.prometheus.port | 9090 | int | Port to expose prometheus metrics |
| pitaya.metrics.tags | map[string]string{} | map[string]string | Tags to be added to reported metrics |

## Concurrency

| Configuration | Default value | Type | Description |
|------------------------------------|---------------|-----------|------------------------------------------|
| pitaya.buffer.agent.messages | 100 | int | Buffer size for received client messages for each agent |
| pitaya.buffer.handler.localprocess | 20 | int | Buffer size for messages received by the handler and processed locally |
| pitaya.buffer.handler.remoteprocess | 20 | int | Buffer size for messages received by the handler and forwarded to remote servers |
| pitaya.concurrency.handler.dispatch | 25 | int | Number of goroutines processing messages at the handler service |

## Modules

These configurations are only used if the modules are created. It is recommended to use Binding Storage module with gRPC RPC service to be able to use all RPC service features.

| Configuration | Default value | Type | Description |
|------------------------------------|---------------|-----------|------------------------------------------|
| pitaya.session.unique | true | bool | Whether Pitaya should enforce unique sessions for the clients, enabling the unique sessions module |
| pitaya.modules.bindingstorage.etcd.endpoints | localhost:2379 | string | Comma separated list of etcd endpoints to be used by the binding storage module, should be the same as the service discovery etcd |
| pitaya.modules.bindingstorage.etcd.prefix | pitaya/ | string | Prefix used for etcd, should be the same as the service discovery |
| pitaya.modules.bindingstorage.etcd.dialtimeout | 5s | time.Time | Timeout to establish the etcd connection |
| pitaya.modules.bindingstorage.etcd.leasettl | 1h | time.Time | Duration of the etcd lease before automatic renewal |

