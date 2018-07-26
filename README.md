# pitaya [![Build Status][7]][8] [![Coverage Status][9]][10] [![GoDoc][1]][2] [![Docs][11]][12] [![Go Report Card][3]][4] [![MIT licensed][5]][6]

[1]: https://godoc.org/github.com/topfreegames/pitaya?status.svg
[2]: https://godoc.org/github.com/topfreegames/pitaya
[3]: https://goreportcard.com/badge/github.com/topfreegames/pitaya
[4]: https://goreportcard.com/report/github.com/topfreegames/pitaya
[5]: https://img.shields.io/badge/license-MIT-blue.svg
[6]: LICENSE
[7]: https://travis-ci.org/topfreegames/pitaya.svg?branch=master
[8]: https://travis-ci.org/topfreegames/pitaya
[9]: https://coveralls.io/repos/github/topfreegames/pitaya/badge.svg?branch=master
[10]: https://coveralls.io/github/topfreegames/pitaya?branch=master
[11]: https://readthedocs.org/projects/pitaya/badge/?version=latest
[12]: https://pitaya.readthedocs.io/en/latest/?badge=latest

Pitaya is an easy to use, fast and lightweight game server framework inspired by [starx](https://github.com/lonnng/starx) and [pomelo](https://github.com/NetEase/pomelo) and built on top of [nano](https://github.com/lonnng/nano)'s networking library.

The goal of pitaya is to provide a basic development framework for distributed multiplayer games and server-side applications.

## How to build a system with `pitaya`

### What does a `pitaya` application look like?

A `pitaya` application is a collection of components made of handlers and/or remotes.

Handlers are methods that will be called directly by the client while remotes are called by other servers via RPCs. Once you register a component to pitaya, pitaya will register to its service container all methods that can be converted to `Handler` or `Remote`.

#### Handlers

Pitaya service handler will be called when the client makes a request and it receives one or two parameters while handling a message:
  - `context.Context`: the context of the request, which contains the client's session.
  - `pointer or []byte`: the payload of the request (optional).

There are two types of handlers, request and push. For the first case the handler must have two return values:
  - `pointer or []byte`: the response payload
  - `error`: an error variable

for the second type the method should not return anything.

#### Remotes

Pitaya service remote will be called by other pitaya servers and it receives one or two parameters while handling the request:
  - `context.Context`: the context of the request.
  - `protobuf`: the payload in protobuf format (optional).

The remote method should always return two parameters:
  - `protobuf`: the response payload in protobuf format.
  - `error`: an error variable

#### Standalone application

The easiest way of running `pitaya` is by starting a standalone application. There's an working example [here](./examples/demo/tadpole).

#### Cluster mode

In order to run several `pitaya` applications in a cluster it is necessary to configure RPC and Service Discovery services. Currently we are using [NATS](https://nats.io/) for RPC and [ETCD](https://github.com/coreos/etcd) for service discovery as default options. The option to use gRPC for RPC is also available. Other options may be implemented in the future.


There's an working example of `pitaya` running in cluster mode [here](./examples/demo/cluster).

To run this example you need to have both nats and etcd running. To start them you can use the following commands:

```bash
docker run -p 4222:4222 -d --name nats-main nats
docker run -d -p 2379:2379 -p 2380:2380 appcelerator/etcd
```

You can start the backend and frontend servers with the following commands:

```make
make run-cluster-example-frontend
make run-cluster-example-backend
```

#### Frontend and backend servers

In short, frontend servers handle client calls while backend servers only handle RPCs coming from other servers. Both types of servers are capable of receiving RPCs.

Frontend servers are responsible for accepting connections with the clients and processing the messages, forwarding them to the appropriate servers as needed. Backend servers receive messages forwarded from the frontend servers. It's possible to set custom forwarding logic based on the client's session data, route and payload.

## Resources

- Documents
  + [API Reference](https://godoc.org/github.com/topfreegames/pitaya)
  + [In-depth documentation](https://pitaya.readthedocs.io/en/latest/)

- Demo
  + [Implement a chat room in ~100 lines with pitaya and WebSocket](./examples/demo/chat) (adapted from [nano](https://github.com/lonnng/nano)'s example)
  + [Tadpole demo](./examples/demo/tadpole) (adapted from [nano](https://github.com/lonnng/nano)'s example)
  + [Pitaya cluster mode example](./examples/demo/cluster)
  + [Pitaya cluster mode with protobuf compression example](./examples/demo/cluster_protobuf)

## Installation

```shell
go get github.com/topfreegames/pitaya

# dependencies
make setup
```

## Testing

```shell
make test
```

## Benchmark

using grpc
```
===============RUNNING BENCHMARK TESTS WITH GRPC===============
--- starting testing servers
--- sleeping for 5 seconds
goos: darwin
goarch: amd64
BenchmarkCreateManyClients-30                                              	    2000	    732922 ns/op
BenchmarkFrontHandlerWithSessionAndRawReturnsRaw-30                        	    2000	    712525 ns/op
BenchmarkFrontHandlerWithSessionAndPtrReturnsPtr-30                        	    2000	    704867 ns/op
BenchmarkFrontHandlerWithSessionAndPtrReturnsPtrManyClientsParallel-30     	    2000	    647892 ns/op
BenchmarkFrontHandlerWithSessionAndPtrReturnsPtrParallel-30                	    2000	    692803 ns/op
BenchmarkFrontHandlerWithSessionOnlyReturnsPtr-30                          	    2000	    880599 ns/op
BenchmarkFrontHandlerWithSessionOnlyReturnsPtrParallel-30                  	    2000	    630234 ns/op
BenchmarkBackHandlerWithSessionOnlyReturnsPtr-30                           	    1000	   1123467 ns/op
BenchmarkBackHandlerWithSessionOnlyReturnsPtrParallel-30                   	    2000	    667119 ns/op
BenchmarkBackHandlerWithSessionOnlyReturnsPtrParallelMultipleClients-30    	    2000	    664865 ns/op
```

using nats
```
===============RUNNING BENCHMARK TESTS WITH NATS===============
--- starting testing servers
--- sleeping for 5 seconds
goos: darwin
goarch: amd64
BenchmarkCreateManyClients-30                                              	    2000	    873214 ns/op
BenchmarkFrontHandlerWithSessionAndRawReturnsRaw-30                        	    2000	    702125 ns/op
BenchmarkFrontHandlerWithSessionAndPtrReturnsPtr-30                        	    2000	    794028 ns/op
BenchmarkFrontHandlerWithSessionAndPtrReturnsPtrManyClientsParallel-30     	    2000	    769600 ns/op
BenchmarkFrontHandlerWithSessionAndPtrReturnsPtrParallel-30                	    2000	    702894 ns/op
BenchmarkFrontHandlerWithSessionOnlyReturnsPtr-30                          	    2000	    984978 ns/op
BenchmarkFrontHandlerWithSessionOnlyReturnsPtrParallel-30                  	    2000	    699000 ns/op
BenchmarkBackHandlerWithSessionOnlyReturnsPtr-30                           	    1000	   1945727 ns/op
BenchmarkBackHandlerWithSessionOnlyReturnsPtrParallel-30                   	    2000	    784496 ns/op
BenchmarkBackHandlerWithSessionOnlyReturnsPtrParallelMultipleClients-30    	    2000	    846923 ns/op
```

## License

[MIT License](./LICENSE)

Forked from [© 2017 nano Authors](https://github.com/lonnng/nano)
