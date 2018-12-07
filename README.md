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

Pitaya is an simple, fast and lightweight game server framework with clustering support and client libraries for iOS, Android, Unity and others through the [C SDK](https://github.com/topfreegames/libpitaya).
It provides a basic development framework for distributed multiplayer games and server-side applications.

## Getting Started

### Prerequisites

* [Go](https://golang.org/) >= 1.10
* [etcd](https://github.com/coreos/etcd) (used for service discovery)
* [nats](https://github.com/nats-io/go-nats) (optional, used for sending and receiving rpc, grpc implementations can be used too if prefered)
* [docker](https://www.docker.com) (optional: used for running etcd and nats dependencies on containers)

### Installing
clone the repo
```
git clone https://github.com/topfreegames/pitaya.git
```
setup pitaya dependencies
```
make setup
```

### Hacking pitaya

Here's one example of running Pitaya:

Start etcd (This command requires docker-compose and will run an etcd container locally. An etcd may be run without docker if prefered.)
```
cd ./examples/testing && docker-compose up -d etcd
```
run the connector frontend server from cluster_grpc example
```
make run-cluster-grpc-example-connector
```
run the room backend server from the cluster_grpc example
```
make run-cluster-grpc-example-room
```

Now there should be 2 pitaya servers running, a frontend connector and a backend room. To send requests, use a REPL client for pitaya [pitaya-cli](https://github.com/topfreegames/pitaya-cli). 

```
$ pitaya-cli
Pitaya REPL Client
>>> connect localhost:3250
connected!
>>> request room.room.entry
>>> sv-> {"code":0,"result":"ok"}
```

## Running the tests
```
make test
```
This command will run both unit and e2e tests.

## Contributing
#TODO

## Authors
* **TFG Co** - Initial work

## License
[MIT License](./LICENSE)

## Acknowledgements
* [nano](https://github.com/lonnng/nano) authors for building the framework pitaya is based on.
* [pomelo](https://github.com/NetEase/pomelo) authors for the inspiration on the distributed design and protocol

## Resources

- Other pitaya-related projects
  + [libpitaya-cluster](https://github.com/topfreegames/libpitaya-cluster)
  + [libpitaya](https://github.com/topfreegames/libpitaya)
  + [pitaya-admin](https://github.com/topfreegames/pitaya-admin)
  + [pitaya-bot](https://github.com/topfreegames/pitaya-bot)
  + [pitaya-cli](https://github.com/topfreegames/pitaya-cli)
  + [pitaya-protos](https://github.com/topfreegames/pitaya-protos)

- Documents
  + [API Reference](https://godoc.org/github.com/topfreegames/pitaya)
  + [In-depth documentation](https://pitaya.readthedocs.io/en/latest/)

- Demo
  + [Implement a chat room in ~100 lines with pitaya and WebSocket](./examples/demo/chat) (adapted from [nano](https://github.com/lonnng/nano)'s example)
  + [Pitaya cluster mode example](./examples/demo/cluster)
  + [Pitaya cluster mode with protobuf protocol example](./examples/demo/cluster_protobuf)

## Benchmarks

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
