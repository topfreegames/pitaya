# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## What this is

Pitaya is a Go game-server framework (module `github.com/topfreegames/pitaya/v3`) with clustering support. The library lives in `pkg/`. The repo root `main.go` + `cmd/` + `repl/` build `pitaya-cli`, a REPL client for talking to Pitaya servers. `xk6-pitaya/` is a k6 load-test extension, joined to the root module through `go.work`.

## Commands

```sh
make setup                  # init git submodules (pitaya-protos) + go get
make build                  # builds ./build/pitaya-cli
make unit-test-coverage     # unit tests only (what CI runs via `make test-coverage`); no docker needed
make e2e-test-nats          # starts etcd+nats via docker compose (examples/testing), builds test server, runs e2e
make e2e-test-grpc          # same, using the gRPC RPC transport (-grpc flag)
make test                   # unit + both e2e suites
make kill-testing-deps      # docker compose down for the test deps
make mocks                  # regenerate gomock mocks (mockgen) into pkg/**/mocks
```

Run a single test:

```sh
go test ./pkg/cluster -run TestNatsRPCServerConfigure -v
```

- `make unit-test-coverage` excludes packages matching examples, constants, mocks, helpers, interfaces, protos, e2e, benchmark (see `TESTABLE_PACKAGES` in the Makefile).
- Some cluster tests start embedded etcd/NATS and leave `127.0.0.1*` / `localhost*` files; `make rm-test-temp-files` cleans them.
- e2e tests run the binary at `examples/testing/server` (built from `examples/testing/main.go`); `-update` forces a rebuild.
- `pkg/protos/*.pb.go` are generated from the `pitaya-protos` git submodule. Do not hand-edit them.
- Mocks are generated; after changing an interface listed under `make mocks` (Agent, Session, NetworkEntity, Pitaya, Serializer, metrics Reporter/Client, Acceptor, worker RPCJob), regenerate the matching mock.

Local cluster demo (needs etcd, e.g. `cd examples/testing && docker compose up -d etcd`):

```sh
make run-cluster-grpc-example-connector   # frontend on :3250
make run-cluster-grpc-example-room        # backend "room" server
```

## Architecture

**Wiring.** `pkg/builder.go` is the composition root. `NewBuilder` creates the default components from `config.PitayaConfig`: etcd service discovery, NATS RPC server/client (cluster mode only), Prometheus/statsd reporters, memory group service, serializer, router, session pool, worker. Callers may swap any exported `Builder` field (e.g. gRPC RPC via `cluster.NewGRPCServer`/`NewGRPCClient`) before `Build()`. `Build()` enforces the mode: `Standalone` must have no SD/RPC; `Cluster` must have all three. It then creates `RemoteService`, `AgentFactory`, and `HandlerService`, and returns the `App` (`pkg/app.go`, interface `Pitaya`).

**Server roles.** Every server has a type (e.g. `connector`, `room`) and is frontend or backend. Only frontends accept client connections, through `acceptor` (TCP/WS). Backends are reached only by RPC.

**Client request flow (frontend).**
1. `acceptor` yields a `PlayerConn`; `HandlerService.Handle` wraps it in an `agent` (one per connection, owns the `session`, heartbeat and write loop). Packets use the Pomelo wire codec (`conn/codec`, `conn/message`).
2. `HandlerService.processMessage` decodes the route `serverType.service.method`. If the server type matches the local server, the message goes to `chLocalProcess`; otherwise to `chRemoteProcess`. Worker goroutines in `Dispatch` drain both channels.
3. Local: handler hooks (`pipeline`) run before/after the handler call, which is resolved through `HandlerPool`.
4. Remote: `RemoteService.remoteProcess` uses `router` to pick a target server (default random of that type via service discovery; custom routes via `app.AddRoute`), then sends via `RPCClient`. The backend's `RPCServer` calls `RemoteService.Call`.

**Handlers vs remotes.** Components (`pkg/component`) are registered with `app.Register` (client-facing handlers) or `app.RegisterRemote` (server-to-server RPC). Methods are discovered by reflection in `component/method.go`: exported, first arg `context.Context`, optional second arg (pointer or `[]byte`). A handler with no return value is a Notify; with return values it is a Request. Remotes must take/return protobuf messages.

**Cross-server session ops.** Backends act on a frontend's user through `networkentity` (`agent_remote.go` on backends). Push, kick, and session bind go back to the owning frontend via `sys` RPCs (`RemoteService.handleRPCSys`, `PushToUser`, `KickUser`, `SessionBindRemote`). Session data set on a backend must be pushed back explicitly (`session.PushToFront`).

**Cluster layer (`pkg/cluster`).** `ServiceDiscovery` (etcd, with lease/heartbeat and a local server cache) plus two RPC transports: NATS (`nats_rpc_*`, subject per server) and gRPC (`grpc_rpc_*`, uses server metadata for address). `InfoRetriever` supplies region/grpc info.

**Context propagation.** `pkg/context` carries values (route, request ID, start time, metric tags, tracing span) across RPC hops inside the request's metadata. Use `pitaya.AddToPropagateCtx` / `GetFromPropagateCtx`, not plain `context.WithValue`, for anything that must cross servers.

**Other packages.** `modules` (lifecycle modules: binary runner, unique session, etc.), `groups` (memory/etcd user groups for broadcast), `worker` (Redis-backed async RPC jobs with retry), `metrics` (Prometheus/statsd reporters and buffer/pool instrumentation), `tracing` (OpenTelemetry), `defaultpipelines` (struct validation hook), `docgenerator` (auto-docs of handlers/remotes).

## Release lines

`main` is the v3 line (`github.com/topfreegames/pitaya/v3`, tagged `v3.0.0-beta.*`). The `v2` branch is the stable v2 line (`v2.11.x`), and it has its own layout (`metrics/`, not `pkg/metrics/`). Many fixes land on both lines as separate PRs (e.g. #500 on `v2` and #501 on `main`). Check which line a change needs to reach.

## Metrics

- Two reporters: Prometheus (`pitaya.metrics.prometheus.enabled`, port 9090) and statsd (`pitaya.metrics.statsd.enabled`, DogStatsD client). Both are off by default. Each metric goes to every enabled reporter (`pkg/metrics/report.go`).
- statsd `ReportSummary` always sends a DogStatsD timer (`TimeInMilliseconds`, `|ms`). Pitaya's summaries are in nanoseconds (`response_time_ns`, `handler_delay_ns`), so Datadog shows them as milliseconds. PR #508 (on `v2`) adds `pitaya.metrics.statsd.summaryashistogram` to send them as `|h`.
- statsd `ReportHistogram` returns `ErrNotImplemented`. A metric that is only a histogram never reaches statsd.
- Difference between the lines: on `v2`, `channel_capacity` is a gauge and `channel_capacity_histogram` is a histogram. On `main`, `channel_capacity` is a histogram and `channel_capacity_histogram` is gone. Statsd users therefore lose channel capacity on v3.
- Prometheus registers a `response_time_ns` histogram with buckets from 1 to 10000. In nanoseconds, that tops out at 10 µs.

## Tracing

`main` uses OpenTelemetry (commits `38dac3f` and `b601443`, July 2024), configured with the standard `OTEL_*` env vars. The `v2` branch still uses OpenTracing/Jaeger.

## Configuration

`pkg/config` wraps Viper. All keys live under `pitaya.` (defaults in `viper_config.go` `fillDefaultValues`, typed structs in `config.go`). Env vars override keys with `.` replaced by `_`, e.g. `PITAYA_METRICS_PROMETHEUS_PORT=9090`. When adding a config field, add it to both the struct default and the defaults map. Full reference: `docs/configuration.rst`.
