# Attention

Pitaya is currently under development and is not yet ready for production use.
We're working on tests and better documentation and we'll update the project as soon as possible.

# pitaya [![GoDoc][1]][2] [![Go Report Card][3]][4] [![MIT licensed][5]][6]

[1]: https://godoc.org/github.com/topfreegames/pitaya?status.svg
[2]: https://godoc.org/github.com/topfreegames/pitaya
[3]: https://goreportcard.com/badge/github.com/topfreegames/pitaya
[4]: https://goreportcard.com/report/github.com/topfreegames/pitaya
[5]: https://img.shields.io/badge/license-MIT-blue.svg
[6]: LICENSE

Pitaya is an easy to use, fast and lightweight game server framework inspired by [starx](https://github.com/lonnng/starx) and [pomelo](https://github.com/NetEase/pomelo) and built on top of [nano](https://github.com/lonnng/nano)'s networking library.

The goal of pitaya is to provide a basic development framework for distributed multiplayer games and server-side applications.

## How to build a system with `pitaya`

#### What does a `pitaya` application look like?

A `pitaya` application is a collection of components made of handlers and/or remotes. Handlers are methods that will be called directly by the client while remotes are called by other servers via RPCs. Once you register a component to pitaya, pitaya will register to its service container all methods that can be converted to `Handler` or `Remote`. Pitaya service handler will be called when the client makes a request and it receives two parameters while handling a message:
  - `*session.Session`: corresponding a client that apply this request or notify.
  - `pointer or []byte`: the payload of the request.

When the server has processed the logic, it must return a struct that will be serialized and sent to the client.

#### Standalone application

The easiest way of running `pitaya` is by starting a standalone application. There's an working example [here](./examples/demo/tadpole).

#### Cluster mode

In order to run several `pitaya` applications in a cluster it is necessary to configure RPC and Service Discovery services. Currently we are using [NATS](https://nats.io/) for RPC and [ETCD](https://github.com/coreos/etcd) for service discovery. Other options may be implemented in the future.


There's an working example of `pitaya` running in cluster mode [here](./examples/demo/cluster). 

To run this example you need to have both nats and etcd running. To start them you can use the following commands:

```bash
docker run -p 4222:4222 -d --name nats-main nats
docker run -d -p 2379:2379 -p 2380:2380 appcelerator/etcd
```

You can start the backend and frontend servers with the following commands:

```make 
make run-cluster-example-frontend
make run-cluster-example-backedn
```

##### Frontend and backend servers

In short, frontend servers handle client calls while backend servers only handle RPCs coming from other servers.

## Resources

- Documents
    + [How to build your first pitaya application](./docs/get_started.md)
    + [Communication protocol](./docs/communication_protocol.md)
    + [API Reference](https://godoc.org/github.com/topfreegames/pitaya)

- Demo
  + [Implement a chat room in ~100 lines with pitaya and WebSocket](./examples/demo/chat) (adpted from [nano](https://github.com/lonnng/nano)'s example)
  + [Tadpole demo](./examples/demo/tadpole) (adpted from [nano](https://github.com/lonnng/nano)'s example)
  + [Pitaya cluster mode example](./examples/demo/cluster)
  + [Pitaya cluster mode with protobuf compression example](./examples/demo/cluster_protobuf)

## Installation

```shell
go get github.com/topfreegames/pitaya

# dependencies
dep ensure
```

## Testing

TBD


## Benchmark

TBD

## License

[MIT License](./LICENSE)

Forked from [© 2017 nano Authors](https://github.com/lonnng/nano)
