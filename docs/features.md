Features
========

Pitaya has a modular and configurable architecture which helps to hide the complexity of scaling the application and managing clients' sessions and communications.

Some of its core features are described below.

## Frontend and backend servers

In cluster mode servers can either be a frontend or backend server.

Frontend servers must specify listeners for receiving incoming client connections. They are capable of forwarding received messages to the appropriate servers according to the routing logic.

Backend servers don't listen for connections, they only receive RPCs, either forwarded client messages (sys rpc) or RPCs from other servers (user rpc).

## Groups

Groups are structures which store information about target users and allows sending broadcast messages to all users in the group and also multicast messages to a subset of the users according to some criteria.

They are useful for creating game rooms for example, you just put all the players from a game room into the same group and then you'll be able to broadcast the room's state to all of them.

## Listeners

Frontend servers must specify one or more acceptors to handle incoming client connections, Pitaya comes with TCP and Websocket acceptors already implemented, and other acceptors can be added to the application by implementing the acceptor interface.

## Acceptor Wrappers

Wrappers can be used on acceptors, like TCP and Websocket, to read and change incoming data before performing the message forwarding. To create a new wrapper just implement the Wrapper interface (or inherit the struct from BaseWrapper) and add it into your acceptor by using the WithWrappers method. Next there are some examples of acceptor wrappers. 

### Rate limiting
Read the incoming data on each player's connection to limit requests troughput. After the limit is exceeded, requests are dropped until slots are available again. The requests count and management is done on player's connection, therefore it happens even before session bind. The used algorithm is the [Leaky Bucket](https://en.wikipedia.org/wiki/Leaky_bucket#Comparison_with_the_token_bucket_algorithm). This algorithm represents a leaky bucket that has its output flow slower than its input flow. It saves each request timestamp in a `slot` (of a total of `limit` slots) and this slot is freed again after `interval`. For example: if `limit` of 1 request in an `interval` of 1 second, when a request happens at 0.2s the next request will only be handled by pitaya after 1s (at 1.2s).

```
0     request
|--------|
   0.2s
0                 available again
|------------------------|
|- 0.2s -|----- 1s ------|
```

## Message forwarding

When a server instance receives a client message, it checks the target server type by looking at the route. If the target server type is different from the receiving server type, the instance forwards the message to an appropriate server instance of the correct type. The client doesn't need to take any action to forward the message, this process is done automatically by Pitaya.

By default the routing function chooses one instance of the target server type at random. Custom functions can be defined to change this behavior.

## Message push

Messages can be pushed to users without previous information about either session or connection status. These push messages have a route (so that the client can identify the source and treat properly), the message, the target ids and the server type the client is expected to be connected to.

## Modules

Modules are entities that can be registered to the Pitaya application and must implement the defined [interface](https://github.com/topfreegames/pitaya/tree/master/interfaces/interfaces.go#L24). Pitaya is responsible for calling the appropriate lifecycle methods as needed, the registered modules can be retrieved by name.

Pitaya comes with a few already implemented modules, and more modules can be implemented as needed. The modules Pitaya has currently are:

### Binary

This module starts a binary as a child process and pipes its stdout and stderr to info and error log messages, respectively.

### Unique session

This module adds a callback for `OnSessionBind` that checks if the id being bound has already been bound in one of the other frontend servers.

### Binding storage

This module implements functionality needed by the gRPC RPC implementation to enable the functionality of broadcasting session binds and pushes to users without knowledge of the servers the users are connected to.

## Monitoring

Pitaya has support for metrics reporting, it comes with Prometheus and Statsd support already implemented and has support for custom reporters that implement the `Reporter` interface. Pitaya also comes with support for open tracing compatible frameworks, allowing the easy integration of Jaeger and others.

The list of metrics reported by the `Reporter` is: 

- Response time: the time to process a message, in nanoseconds. It is segmented
  by route, status, server type and response code;
- Process delay time: the delay to start processing a message, in nanoseconds;
  It is segmented by route and server type;
- Exceeded Rate Limit: the number of blocked requests by exceeded rate limiting;
- Connected clients: number of clients connected at the moment;
- Server count: the number of discovered servers by service discovery. It is
  segmented by server type;
- Channel capacity: the available capacity of the channel;
- Dropped messages: the number of rpc server dropped messages, that is, messages that are not handled;
- Goroutines count: the current number Goroutines;
- Heap size: the current heap size;
- Heap objects count: the current number of objects at the heap;
- Worker jobs retry: the current amount of RPC reliability worker job retries; 
- Worker jobs total: the current amount of RPC reliability worker jobs. It is
  segmented by job status;
- Worker queue size: the current size of RPC reliability worker job queues. It
  is segmented by each available queue.

### Custom Metrics

Besides pitaya default monitoring, it is possible to create new metrics. If using only Statsd reporter, no configuration is needed. If using Prometheus, it is necessary do add a configuration specifying the metrics parameters. More details on [doc](configuration.html#metrics-reporting) and this [example](https://github.com/topfreegames/pitaya/tree/master/examples/demo/custom_metrics).

## Pipelines

Pipelines are middlewares which allow methods to be executed before and after handler requests, they receive the request's context and request data and return the request data, which is passed to the next method in the pipeline.

## RPCs

Pitaya has support for RPC calls when in cluster mode, there are two components to enable this, RPC client and RPC server. There are currently two options for using RPCs implemented for Pitaya, NATS and gRPC, the default is NATS.

There are two types of RPCs, _Sys_ and _User_.

### Sys RPCs

These are the RPCs done by the servers when forwarding handler messages to the appropriate server type.

### User RPCs

User RPCs are done when the application actively calls a remote method in another server. The call can specify the ID of the target server or let Pitaya choose one according to the routing logic.

### User Reliable RPCs

These are done when the application calls a remote using workers, that is, Pitaya retries the RPC if any error occurrs.

**Important**: the remote that is being called must be idempotent; also the ReliableRPC will not return the remote's reply since it is asynchronous, it only returns the job id (jid) if success.

## Server operation mode

Pitaya has two types of operation: standalone and cluster mode.

### Standalone mode

In standalone mode the servers don't interact with one another, don't use service discovery and don't have support to RPCs. This is a limited version of the framework which can be used when the application doesn't need to have different types of servers or communicate among them.

### Cluster mode

Cluster mode is a more complete mode, using service discovery, RPC client and server and remote communication among servers of the application. This mode is useful for more complex applications, which might benefit from splitting the responsabilities among different specialized types of servers. This mode already comes with default services for RPC calls and service discovery.

## Serializers

Pitaya has support for different types of message serializers for the messages sent to and from the client, the default serializer is the JSON serializer and Pitaya comes with native support for the Protobuf serializer as well. New serializers can be implemented by implementing the `serialize.Serializer` interface.

The desired serializer can be set by the application by calling the `SetSerializer` method from the `pitaya` package.

## Service discovery

Servers operating in cluster mode must have a service discovery client to be able to work. Pitaya comes with a default client using etcd, which is used if no other client is defined. The service discovery client is responsible for registering the server and keeping the list of valid servers updated, as well as providing information about requested servers as needed.

## Sessions

Every connection established by the clients has an associated session instance, which is ephemeral and destroyed when the connection closes. Sessions are part of the core functionality of Pitaya, because they allow asynchronous communication with the clients and storage of data between requests. The main features of sessions are:

* **ID binding** - Sessions can be bound to an user ID, allowing other parts of the application to send messages to the user without needing to know which server or connection the user is connected to
* **Data storage** - Sessions can be used for data storage, storing and retrieving data between requests
* **Message passing** - Messages can be sent to connected users through their sessions, without needing to have knowledge about the underlying connection protocol
* **Accessible on requests** - Sessions are accessible on handler requests in the context instance
* **Kick** - Users can be kicked from the server through the session's `Kick` method

Even though sessions are accessible on handler requests both on frontend and backend servers, their behavior is a bit different if they are a frontend or backend session. This is mostly due to the fact that the session actually lives in the frontend servers, and just a representation of its state is sent to the backend server.

A session is considered a frontend session if it is being accessed from a frontend server, and a backend session is accessed from a backend server. Each kind of session is better described below.

### Frontend sessions

Sessions are associated to a connection in the frontend server, and can be retrieved by session ID or bound user ID in the server the connection was established, but cannot be retrieved from a different server.

Callbacks can be added to some session lifecycle changes, such as closing and binding. The callbacks can be on a per-session basis (with `s.OnClose`) or for every session (with `OnSessionClose`, `OnSessionBind` and `OnAfterSessionBind`).

### Backend sessions

Backend sessions have access to the sessions through the handler's methods, but they have some limitations and special characteristics. Changes to session variables must be pushed to the frontend server by calling `s.PushToFront` (this is not needed for `s.Bind` operations), setting callbacks to session lifecycle operations is also not allowed. One can also not retrieve a session by user ID from a backend server.

