# Pitaya Game Server Framework

## Development Commands

### Setup
```bash
make setup          # Install Go dependencies
make setup-ci       # Install CI tools (goveralls, gocovmerge)
```

### Testing
```bash
make test           # Run all tests (unit + e2e)
make test-coverage  # Unit tests with coverage
make e2e-test       # End-to-end tests (requires docker deps)
make e2e-test-nats  # E2E tests with NATS RPC
make e2e-test-grpc  # E2E tests with gRPC RPC
```

### Dependencies
E2E tests require docker-compose services:
```bash
cd examples/testing && docker compose up -d  # Start etcd, nats, redis
```

### Code Generation
```bash
make protos-compile  # Compile protobuf files
make mocks           # Generate all mocks (agent, session, etc.)
```

## Architecture

### Module Structure
- `github.com/topfreegames/pitaya/v3` - Main framework (pkg/)
- `github.com/topfreegames/pitaya/xk6-pitaya` - k6 extension (xk6-pitaya/)

### Key Packages
- `pkg/app.go` - Main application interface
- `pkg/cluster/` - Distributed server support (etcd, nats, grpc)
- `pkg/component/` - Service registration and handlers
- `pkg/session/` - Client session management
- `pkg/worker/` - Background job processing

### Server Types
- Frontend servers (connector) - Handle client connections
- Backend servers (room, game) - Handle game logic
- Worker servers - Process background jobs

## Testing Notes

### Unit Test Exclusions
Tests exclude these packages: `examples`, `constants`, `mocks`, `helpers`, `interfaces`, `protos`, `e2e`, `benchmark`

### E2E Test Flow
1. Start docker dependencies: `cd examples/testing && docker compose up -d`
2. Build test server: `make ensure-testing-bin`
3. Run tests: `make e2e-test`
4. Cleanup: `make kill-testing-deps`

### Mock Generation
Uses `mockgen` from `github.com/golang/mock`. Run `make mocks` after interface changes.

## CI/CD

- Branches: `main`, `v2`
- Go version: 1.25
- Runs: unit tests, e2e-nats, e2e-grpc
