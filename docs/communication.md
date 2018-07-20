Communication
=============

In this section we will describe in detail the communication process between the client and the server. From establishing the connection, sending a request and receiving a response. The example is going to assume the application is running in cluster mode and that the target server is not the same as the one the client is connected to.


## Establishing the connection

The overview of the connection flow is:

* Establish low level connection with acceptor
* Pass the connection to the handler service
* Handler service creates a new agent for the connection
* Handler service reads message from the connection
* Message is decoded with configured decoder
* Decoded packet from the message is processed
* First packet must be a handshake request, to which the server returns a handshake response with the serializer, route dictionary and heartbeat timeout
* Client must then reply with a handshake ack, connection is then established
* Following requests can be of type data or heartbeat, heartbeat just keeps the connection alive
* Data messages are processed by the handler and the target server type is extracted from the message route, the message is deserialized using the specified method
* If the target server type is the same as the current server, the current server calls the appropriate method to handle the request
* If the target server type is different from the current server, the server makes a remote call to the right type of server, selecting one server according to the routing function logic. The remote call includes the current representation of the client's session
* The receiving remote server receives the request and handles it as a _Sys_ RPC call, creating a new remote agent to handle the request, this agent receives the session's representation
* The appropriate handler message is then called by the remote server, which deals with the response and returns it to the frontend server
* If the backend server wants to modify the session it needs to modify and push the modifications to the frontend server explicitly
* Once the frontend server receives the response it forwards the message to the session specifying the request message ID
* The agent receives the requests, serializes it and sends to the low-leve connection

### Acceptors

The first thing the client must do is establish a connection with the Pitaya server. And for that to happen, the server must have specified one or more acceptors.

Acceptors are the entities responsible for listening for connections, establishing them, abstracting and forwarding them to the handler service. Pitaya comes with support for TCP and websocket acceptors. Custom acceptors can be implemented and added to Pitaya applications, they just need to implement the proper interface.

### Handler service

After the low level connection is established it is passed to the handler service to handle. The handler service


*TODO*
Agent

Handshake

Forwarder

Serializer

Handler
