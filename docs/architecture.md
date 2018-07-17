Architecture
============

Pitaya has a modular and configurable architecture which helps to hide the complexity of scaling the application and managing clients' sessions and communications.

## Standalone mode

Pitaya has two types of operation: standalone and cluster mode.

In standalone mode the servers don't interact with one another, don't use service discovery and don't have support to RPCs. This is a limited version of the framework which can be used when the application doesn't need to have different types of servers or communicate among them.

## Cluster mode

Cluster mode is a more complete mode, using service discovery, RPC client and server and remote communication among servers of the application. This mode is useful for more complex applications, which might benefit from splitting the responsabilities among different specialized types of servers. This mode already comes with default services for RPC calls and service discovery.

### Frontend and backend servers

In cluster mode servers can either be a frontend or backend server.

Frontend servers must specify listeners and receive incoming client connections. They forward received messages to the appropriate servers according to the routing logic.

Backend servers don't listen for connections, they only receive RPCs, either forwarded client messages or RPCs from other servers.

### Listeners

Frontend servers must specify one or more acceptors to handle incoming client connections, Pitaya comes with TCP and Websocket acceptors already implemented, and other acceptors can be added to the application by implementing the acceptor interface.

### Message forwarding

When a server instance receives a client message, it checks the target server type by looking at the route. If the target server type is different from the receiving server type, the instance forwards the message to an appropriate server instance of the correct type. The client doesn't need to take any action to forward the message, this process is done automatically by Pitaya.

By default the routing function chooses one instance of the target server type at random. Custom functions can be defined to change this behavior.

### Sessions

Every connection established by the clients has an associated session instance, the session is destroyed when the connection closes. The session can be bound to an user id and can be used to set or retrieve data. The session is passed to every handler method through the context instance. Sessions can be used to send push messages to the client or kick her.

There are some differences if the server is a frontend or backend server. Explained below

#### Frontend sessions

Sessions are associated to a connection in the frontend server, and can be retrieved by session ID or bound user ID in the server the connection was established, but cannot be retrieved from a different server. Callbacks can be added to some session lifecycle changes, such as closing and binding. The callbacks can be on a per-session basis (with `s.OnClose`) or for every session (with `OnSessionClose`, `OnSessionBind` and `OnAfterSessionBind`).

#### Backend sessions

Backend sessions have access to the sessions through the handler's methods, but they have some limitations and special characteristics. Changes to session variables must be pushed to the frontend server by calling `s.PushToFront` (this is not needed for `s.Bind` operations), setting callbacks to session lifecycle operations is also not allowed. One can also not retrieve a session by user ID from a backend server.

### Service discovery

Servers operating in cluster mode must have a service discovery client to be able to work, Pitaya comes with a default client using etcd, which is used if no other client is defined. The service discovery client is responsible for registering the server and keeping the list of valid server updated, as well as providing information about requested servers as needed.

### RPCs

Pitaya has support for RPC calls when in cluster mode, there are two components to enable this, RPC client and RPC server. There are currently two options for using RPCs implemented for Pitaya, NATS and gRPC, the default is NATS.

There are two types of RPCs, _Sys_ and _User_.

#### Sys RPCs

These are the RPCs done by the servers when forwarding handler messages to the appropriate server type.

#### User RPCs

User RPCs are done when the application actively calls a remote method in another server. The call can specify the ID of the target server or let Pitaya choose one according to the routing logic.

### Modules

Modules are entities that can be registered to the Pitaya application and must implement the defined interface. Pitaya is responsible for calling the appropriate lifecycle methods as needed, the registered modules can be retrieved by name.

Pitaya comes with a few already implemented modules, and more modules can be implemented as needed. The modules Pitaya has currently are:

#### Binary

This module starts a binary as a child process and pipes its stdout and stderr to info and error log messages, respectively.

#### Unique session

This module adds a callback for `OnSessionBind` that checks if the id being bound has already been bound in one of the other frontend servers.

#### Binding storage

This module implements functionality needed by the gRPC RPC implementation to enable the functionality of broadcasting session binds and pushes to users without knowledge of the servers the users are connected to.
