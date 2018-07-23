Communication
=============

In this section we will describe in detail the communication process between the client and the server. From establishing the connection, sending a request and receiving a response. The example is going to assume the application is running in cluster mode and that the target server is not the same as the one the client is connected to.


## Establishing the connection

The overview of what happens when a client connects and makes a request is:

* Establish low level connection with acceptor
* Pass the connection to the handler service
* Handler service creates a new agent for the connection
* Handler service reads message from the connection
* Message is decoded with configured decoder
* Decoded packet from the message is processed
* First packet must be a handshake request, to which the server returns a handshake response with the serializer, route dictionary and heartbeat timeout
* Client must then reply with a handshake ack, connection is then established
* Data messages are processed by the handler and the target server type is extracted from the message route, the message is deserialized using the specified method
* If the target server type is different from the current server, the server makes a remote call to the right type of server, selecting one server according to the routing function logic. The remote call includes the current representation of the client's session
* The receiving remote server receives the request and handles it as a _Sys_ RPC call, creating a new remote agent to handle the request, this agent receives the session's representation
* The before pipeline functions are called and the handler message is deserialized
* The appropriate handler is then called by the remote server, which returns the response that is then serialized and the after pipeline functions are executed
* If the backend server wants to modify the session it needs to modify and push the modifications to the frontend server explicitly
* Once the frontend server receives the response it forwards the message to the session specifying the request message ID
* The agent receives the requests, encodes it and sends to the low-level connection

### Acceptors

The first thing the client must do is establish a connection with the Pitaya server. And for that to happen, the server must have specified one or more acceptors.

Acceptors are the entities responsible for listening for connections, establishing them, abstracting and forwarding them to the handler service. Pitaya comes with support for TCP and websocket acceptors. Custom acceptors can be implemented and added to Pitaya applications, they just need to implement the proper interface.

### Handler service

After the low level connection is established it is passed to the handler service to handle. The handler service is responsible for handling the lifecycle of the clients' connections. It reads from the low-level connection, decodes the received packets and handles them properly, calling the local server's handler if the target server type is the same as the local one or forwarding the message to the remote service otherwise.

Pitaya has a configuration to define the number of concurrent messages being processed at the same time, both local and remote messages count for the concurrency, so if the server expects to deal with slow routes this configuration might need to be tweaked a bit. The configuration is `pitaya.concurrency.handler.dispatch`.

### Agent

The agent entity is responsible for storing information about the client's connection, it stores the session, encoder, serializer, state, connection, among others. It is used to communicate with the client to send messages and also ensure the connection is kept alive.

### Route compression

The application can define a dictionary of compressed routes before starting, these routes are sent to the clients on the handshake. Compressing the routes might be useful for the routes that are used a lot to reduce the communication overhead.

### Handshake

The first operation that happens when a client connects is the handshake. The handshake is initiated by the client, who sends informations about the client, such as platform, version of the client library, and others, and can also send user data in this step. This data is stored in the client's session and can be accessed later. The server replies with heartbeat interval, name of the serializer and the dictionary of compressed routes.

### Remote service

The remote service is responsible both for making RPCs and for receiving and handling them. In the case of a forwarded client request the RPC is of type _Sys_.

In the calling side the service is responsible for identifying the proper server to be called, both by server type and by routing logic.

In the receiving side the service identifies it is a _Sys_ RPC and creates a remote agent to handle the request. This remote agent is short-lived, living only while the request is alive, changes to the backend session do not automatically reflect in the associated frontend session, they need to be explicitly committed by pushing them. The message is then forwarded to the appropriate handler to be processed.

### Pipeline

The pipeline in Pitaya is a set of functions that can be defined to be run before or after every handler request. The functions receive the context and the raw message and should return the request object and error, they are allowed to modify the context and return a modified request. If the before function returns an error the request fails and the process is aborted.

### Serializer

The handler must first deserialize the message before processing it. So the function responsible for calling the handler method first deserializes the message, calls the method and then serializes the response returned by the method and returns it back to the remote service.

### Handler

Each Pitaya server can register multiple handler structures, as long as they have different names. Each structure can have multiple methods and Pitaya will choose the right structure and methods based on the called route.
