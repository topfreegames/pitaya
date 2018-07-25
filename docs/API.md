Pitaya API
==========

## Handlers

Handlers are one of the core features of Pitaya, they are the entities responsible for receiving the requests from the clients and handling them, returning the response if the method is a request handler, or nothing, if the method is a notify handler.

### Signature

Handlers must be public methods of the struct and have a signature following:

Arguments
* `context.Context`: the context of the request, which contains the client's session.
* `pointer or []byte`: the payload of the request (_optional_).

Notify handlers return nothing, while request handlers must return:
* `pointer or []byte`: the response payload
* `error`: an error variable


### Registering handlers

Handlers must be explicitly registered by the application by calling `pitaya.Register` with a instance of the handler component. The handler's name can be defined by calling `pitaya/component`.WithName(`"handlerName"`) and the methods can be renamed by using `pitaya/component`.WithNameFunc(`func(string) string`).

The clients can call the handler by calling `serverType.handlerName.methodName`.


### Routing messages

Messages are forwarded by pitaya to the appropriate server type, and custom routers can be added to the application by calling `pitaya.AddRoute`, it expects two arguments:

* `serverType`: the server type of the target requests to be routed
* `routingFunction`: the routing function with the signature `func(*session.Session, *route.Route, []byte, map[string]*cluster.Server) (*cluster.Server, error)`, it receives the user's session, the route being requested, the message and the map of valid servers of the given type, the key being the servers' ids

The server will then use the routing function when routing requests to the given server type.


### Lifecycle Methods

Handlers can optionally implement the following lifecycle methods:

* `Init()` - Called by Pitaya when initializing the application
* `AfterInit()` - Called by Pitaya after initializing the application
* `BeforeShutdown()` - Called by Pitaya when shutting down components, but before calling shutdown
* `Shutdown()` - Called by Pitaya after the start of shutdown


### Handler example

Below is a very barebones example of a handler definition, for a complete working example, check the [cluster demo](https://github.com/topfreegames/pitaya/tree/master/examples/demo/cluster).

```go
import (
  "github.com/topfreegames/pitaya"
  "github.com/topfreegames/pitaya/component"
)

type Handler struct {
  component.Base
}

type UserRequestMessage struct {
  Name    string `json:"name"`
  Content string `json:"content"`
}

type UserResponseMessage {
}

type UserPushMessage{
  Command string `json:"cmd"`
}

// Init runs on service initialization (not required to be defined)
func (h *Handler) Init() {}

// AfterInit runs after initialization (not required to be defined)
func (h *Handler) AfterInit() {}

// TestRequest can be called by the client by calling <servertype>.testhandler.testrequest
func (h *Handler) TestRequest(ctx context.Context, msg *UserRequestMessage) (*UserResponseMessage, error) {
  return &UserResponseMessage{}, nil
}

func (h *Handler) TestPush(ctx context.Context, msg *UserPushMessage) {
}

func main() {
  pitaya.Register(
    &Handler{}, // struct to register as handler
    component.WithName("testhandler"), // name of the handler, used by the clients
    component.WithNameFunc(strings.ToLower), // naming conversion scheme to be used by the clients
  )

  ...
}

```

## Remotes

Remotes are one of the core features of Pitaya, they are the entities responsible for receiving the RPCs from other Pitaya servers.

### Signature

Remotes must be public methods of the struct and have a signature following:

Arguments
* `context.Context`: the context of the request.
* `proto.Message`: the payload of the request (_optional_).

Remote methods must return:
* `proto.Message`: the response payload in protobuf format
* `error`: an error variable


### Registering remotes

Remotes must be explicitly registered by the application by calling `pitaya.RegisterRemote` with a instance of the remote component. The remote's name can be defined by calling `pitaya/component`.WithName(`"remoteName"`) and the methods can be renamed by using `pitaya/component`.WithNameFunc(`func(string) string`).

The servers can call the remote by calling `serverType.remoteName.methodName`.


### RPC calls

There are two options when sending RPCs between servers:
* **Specify only server type**: In this case Pitaya will select one of the available servers at random
* **Specify server type and ID**: In this scenario Pitaya will send the RPC to the specified server


### Lifecycle Methods

Remotes can optionally implement the following lifecycle methods:

* `Init()` - Called by Pitaya when initializing the application
* `AfterInit()` - Called by Pitaya after initializing the application
* `BeforeShutdown()` - Called by Pitaya when shutting down components, but before calling shutdown
* `Shutdown()` - Called by Pitaya after the start of shutdown

### Remote example

For a complete working example, check the [cluster demo](https://github.com/topfreegames/pitaya/tree/master/examples/demo/cluster).
