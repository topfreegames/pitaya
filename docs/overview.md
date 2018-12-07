Overview
========

Pitaya is an easy to use, fast and lightweight game server framework inspired by [starx](https://github.com/lonnng/starx) and [pomelo](https://github.com/NetEase/pomelo) and built on top of [nano](https://github.com/lonnng/nano)'s networking library.

The goal of pitaya is to provide a basic, robust development framework for distributed multiplayer games and server-side applications.

## Features

* **User sessions** - Pitaya has support for user sessions, allowing binding sessions to user ids, setting custom data and retrieving it in other places while the session is active
* **Cluster support** - Pitaya comes with support to default service discovery and RPC modules, allowing communication between different types of servers with ease
* **WS and TCP listeners** - Pitaya has support for TCP and Websocket acceptors, which are abstracted from the application receiving the requests
* **Handlers and remotes** - Pitaya allows the application to specify its handlers, which receive and process client messages, and its remotes, which receive and process RPC server messages. They can both specify custom init, afterinit and shutdown methods
* **Message forwarding** - When a server receives a handler message it forwards the message to the server of the correct type
* **Client library SDK** - [libpitaya](https://github.com/topfreegames/libpitaya) is the official client library SDK for Pitaya
* **Monitoring** - Pitaya has support for Prometheus and statsd by default and accepts other custom reporters that implement the Reporter interface
* **Open tracing compatible** - Pitaya is compatible with [open tracing](http://opentracing.io/), so using [Jaeger](https://github.com/jaegertracing/jaeger) or any other compatible tracing framework is simple
* **Custom modules** - Pitaya already has some default modules and supports custom modules as well
* **Custom serializers** - Pitaya natively supports JSON and Protobuf messages and it is possible to add other custom serializers as needed
* **Write compatible servers in other languages** - Using [libpitaya-cluster](https://github.com/topfreegames/libpitaya-cluster) its possible to write pitaya-compatible servers in other languages that are able to register in the cluster and handle RPCs, there's already a csharp library that's compatible with unity and a WIP of a python library in the repo.
* **REPL Client for development/debugging** - [Pitaya-cli](https://github.com/topfreegames/pitaya-cli) is a REPL client that can be used for making development and debugging of pitaya servers easier.
* **Bots for integration/stress tests** - [Pitaya-bot](https://github.com/topfreegames/pitaya-bot) is a server test framework that can easily copy users behaviour to test corner case scenarios, which can validate the responses received, or make massive accesses into pitaya servers. 

## Architecture

Pitaya was developed considering modularity and extendability at its core, while providing solid basic functionalities to abstract client interactions to well defined interfaces. The full API documentation is available in Godoc format at [godoc](https://godoc.org/github.com/topfreegames/pitaya).

## Who's Using it

Well, right now, only us at TFG Co, are using it, but it would be great to get a community around the project. Hope to hear from you guys soon!

## How To Contribute?

Just the usual: Fork, Hack, Pull Request. Rinse and Repeat. Also don't forget to include tests and docs (we are very fond of both).
