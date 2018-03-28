# Attention

Pitaya is currently under development and is not yet ready for production use.
We're working on tests and better documentation and we'll update the project as soon as possible.

# pitaya [![Build Status][1]][2] [![GoDoc][3]][4] [![Go Report Card][5]][6] [![MIT licensed][7]][8]

[1]: https://travis-ci.org/topfreegames/pitaya.svg?branch=master
[2]: https://travis-ci.org/topfreegames/pitaya
[3]: https://godoc.org/github.com/topfreegames/pitaya?status.svg
[4]: https://godoc.org/github.com/topfreegames/pitaya
[5]: https://goreportcard.com/badge/github.com/topfreegames/pitaya
[6]: https://goreportcard.com/report/github.com/topfreegames/pitaya
[7]: https://img.shields.io/badge/license-MIT-blue.svg
[8]: LICENSE

pitaya is an easy to use, fast, lightweight game server networking library for Go built on top of [nano](https://github.com/lonnng/nano).
It provides a core network architecture and a series of tools and libraries that
can help developers eliminate boring duplicate work for common underlying logic.
The goal of pitaya is to improve development efficiency by eliminating the need to
spend time on repetitious network related programming.

pitaya was designed for server-side applications like real-time games, social games,
mobile games, etc of all sizes.

## How to build a system with `pitaya`

#### What does a `pitaya` application look like?

In fact, the `pitaya` application is a collection of  [Component ](./docs/get_started.md#component) , and a component is a bundle of  [Handler](./docs/get_started.md#handler), once you register a component to pitaya, pitaya will register all methods that can be converted to `Handler` to pitaya service container. Service is accessed by `Component.Handler`, and the handler will be called when the client makes a request. The handler receives two parameters while handling a message:
  - `*session.Session`: corresponding a client that apply this request or notify.
  - `*protocol.FooBar`: the payload of the request.

When the server has processed the logic, it can respond or push message to the client by `session.Response(payload)` and `session.Push('eventName', payload)`, or return error if some unexpected error happens.

#### How to build distributed system with `pitaya`

pitaya has no built-in distributed system components, but you can easily implement it with `gRPC` and `smux` . Here we take grpc as an example.

- First of all, you need to define a remote component
```go
type RemoteComponent struct {
	rpcClients []*grpc.ClientConn
}
```

- Second, fetch all grpc servers infomation from services like `etcd` or `consul`  in `pitaya` lifetime hooks
```go
type ServerInfo struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// lifetime callback
func (r *RemoteComponent) Init() {
	// fetch server list from etcd
	resp, err := http.Get("http://your_etcd_server/backend/server_list/area/10023")
	if err != nil {
		panic(err)
	}

	servers := []ServerInfo{}
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		panic(err)
	}

	for i := range servers {
		server := servers[i]
		client, err := grpc.Dial(fmt.Sprintf("%s:%d", server.Host, server.Post), options)
		if err != nil {
			panic(err)
		}
		r.rpcClients = append(r.rpcClients, client)
	}
}

func (r *RemoteComponent) client(s *session.Session) *grpc.ClientConn {
	// load balance
	return r.rpcClients[s.UID() % len(s.rpcClients)]
}

// Your handler, accessed by:
// client.Request("RemoteComponent.DemoHandler", &pb.DemoMsg{/*...*/})
func (r *RemoteComponent) DemoHandler(s *session.Session, msg *pb.DemoMsg) error {
	client := r.client(s)
	// do something with client
	// ....
	// ...
	return nil
}
```

The pitaya will remain simple, but you can perform any operations in the component and get the desired goals. You can startup a group of `pitaya` application as agent to dispatch message to backend servers.

## Documents

- English
    + [How to build your first pitaya application](./docs/get_started.md)
    + [Route compression](./docs/route_compression.md)
    + [Communication protocol](./docs/communication_protocol.md)
    + [Design patterns](./docs/design_patterns.md)
    + [API Reference(Server)](https://godoc.org/github.com/topfreegames/pitaya)
    + [How to integrate `Lua` into `pitaya` component(incomplete)](.)

## Resources

- Javascript
  + [pitaya-websocket-client](https://github.com/topfreegames/pitaya-websocket-client)
  + [pitaya-egret-client](https://github.com/topfreegames/pitaya-egret-client)

- Demo
  + [Implement a chat room in 100 lines with pitaya and WebSocket](./examples/demo/chat)
  + [Tadpole demo](./examples/demo/tadpole)


## Installation

```shell
go get github.com/topfreegames/pitaya

# dependencies
dep ensure
```

## Benchmark

```shell
cd $GOPATH/src/github.com/topfreegames/pitaya/benchmark/io
go test -v -tags "benchmark"
```

## License

[MIT License](./LICENSE)
