
ifeq ($(OS), Windows_NT)
	BIN := pitaya.exe
	MKFOLDER := if not exist "build" mkdir build
	GREP_CMD := findstr /V
else
	BIN := pitaya
	MKFOLDER := mkdir -p build
	GREP_CMD := grep -v
endif

TESTABLE_PACKAGES = $(shell go list ./... | $(GREP_CMD) examples | $(GREP_CMD) constants | $(GREP_CMD) mocks | $(GREP_CMD) helpers | $(GREP_CMD) interfaces | $(GREP_CMD) protos | $(GREP_CMD) e2e | $(GREP_CMD) benchmark)

.PHONY: build

build:
	@$(MKFOLDER)
	@go build -o build/$(BIN)

setup: init-submodules
	@go get ./...

init-submodules:
	@git submodule init

setup-ci:
	@go get github.com/mattn/goveralls
	@go get -u github.com/wadey/gocovmerge

setup-protobuf-macos:
	@brew install protobuf
	@go get github.com/golang/protobuf/protoc-gen-go

run-chat-example:
	@cd examples/testing && docker-compose up -d etcd nats && cd ../demo/chat/ && go run main.go

run-cluster-example-frontend:
	@PITAYA_METRICS_PROMETHEUS_PORT=9090 go run examples/demo/cluster/main.go

run-cluster-protobuf-frontend-example:
	@cd examples/demo/cluster_protobuf && go run main.go

run-cluster-protobuf-backend-example:
	@cd examples/demo/cluster_protobuf && go run main.go --port 3251 --type room --frontend=false

run-cluster-example-backend:
	@PITAYA_METRICS_PROMETHEUS_PORT=9091 go run examples/demo/cluster/main.go --port 3251 --type room --frontend=false

run-cluster-grpc-example-connector:
	@cd examples/demo/cluster_grpc && go run main.go

run-cluster-grpc-example-room:
	@cd examples/demo/cluster_grpc && go run main.go --port 3251 --rpcsvport 3435 --type room --frontend=false

run-cluster-worker-example-room:
	@cd examples/demo/worker && go run main.go --type room --frontend=true

run-cluster-worker-example-metagame:
	@cd examples/demo/worker && go run main.go --type metagame --frontend=false

run-cluster-worker-example-worker:
	@cd examples/demo/worker && go run main.go --type worker --frontend=false

run-custom-metrics-example:
	@cd examples/demo/custom_metrics && go run main.go --port 3250

run-rate-limiting-example:
	@go run examples/demo/rate_limiting/main.go

protos-compile-demo:
	@protoc -I examples/demo/protos examples/demo/protos/*.proto --go_out=.

protos-compile:
	@cd benchmark/testdata && ./gen_proto.sh
	@protoc -I pitaya-protos/ pitaya-protos/*.proto --go_out=./pkg/protos/ --go_opt paths=source_relative --go-grpc_out ./pkg/protos/ --go-grpc_opt paths=source_relative --go-grpc_opt require_unimplemented_servers=false
	@protoc -I pitaya-protos/test pitaya-protos/test/*.proto --go_out ./pkg/protos/test/ --go_opt paths=source_relative
	@protoc -I examples/demo/protos/ examples/demo/protos/*.proto --go_out ./examples/demo/protos --go_opt paths=source_relative

rm-test-temp-files:
	@rm -f cluster/127.0.0.1* 127.0.0.1*
	@rm -f cluster/localhost* localhost*

ensure-testing-bin:
	@[ -f ./examples/testing/server ] || go build -o ./examples/testing/server ./examples/testing/main.go

ensure-testing-deps:
	@cd ./examples/testing && docker-compose up -d

ensure-e2e-deps-grpc:
	@cd ./examples/testing && docker-compose up -d etcd

kill-testing-deps:
	@cd ./examples/testing && docker-compose down; true

e2e-test: e2e-test-nats e2e-test-grpc

e2e-test-nats: ensure-testing-deps ensure-testing-bin
	@echo "===============RUNNING E2E NATS TESTS==============="
	@go test ./e2e/e2e_test.go -update

e2e-test-grpc: ensure-testing-deps ensure-testing-bin
	@echo "===============RUNNING E2E GRPC TESTS==============="
	@go test ./e2e/e2e_test.go -update -grpc

bench-nats-sv:
	@PITAYA_METRICS_PROMETHEUS_PORT=9098 ./examples/testing/server -type game -frontend=false > /dev/null 2>&1 & echo $$! > back.PID
	@PITAYA_METRICS_PROMETHEUS_PORT=9099 ./examples/testing/server -type connector -frontend=true > /dev/null 2>&1 & echo $$! > front.PID

bench-grpc-sv:
	@PITAYA_METRICS_PROMETHEUS_PORT=9098 ./examples/testing/server -grpc -grpcport=3435 -type game -frontend=false > /dev/null 2>&1 & echo $$! > back.PID
	@PITAYA_METRICS_PROMETHEUS_PORT=9099 ./examples/testing/server -grpc -grpcport=3436 -type connector -frontend=true > /dev/null 2>&1 & echo $$! > front.PID

benchmark-test-nats: ensure-testing-deps ensure-testing-bin
	@echo "===============RUNNING BENCHMARK TESTS WITH NATS==============="
	@echo "--- starting testing servers"
	@echo "--- sleeping for 5 seconds"
	@make bench-nats-sv
	@sleep 5
	@go test ./benchmark/benchmark_test.go -bench=.
	@echo "--- killing testing servers"
	@kill `cat back.PID` && rm back.PID
	@kill `cat front.PID` && rm front.PID

benchmark-test-grpc: ensure-e2e-deps-grpc ensure-testing-bin
	@echo "===============RUNNING BENCHMARK TESTS WITH GRPC==============="
	@echo "--- starting testing servers"
	@echo "--- sleeping for 5 seconds"
	@make bench-grpc-sv
	@sleep 5
	@go test ./benchmark/benchmark_test.go -bench=.
	@echo "--- killing testing servers"
	@kill `cat back.PID` && rm back.PID
	@kill `cat front.PID` && rm front.PID

unit-test-coverage: kill-testing-deps
	@echo "===============RUNNING UNIT TESTS==============="
	@go test $(TESTABLE_PACKAGES) -coverprofile coverprofile.out

test: kill-testing-deps test-coverage
	@make rm-test-temp-files
	@make ensure-testing-deps
	@sleep 10
	@make e2e-test

test-coverage: unit-test-coverage
	@make rm-test-temp-files

test-coverage-html: test-coverage
	@go tool cover -html=coverprofile.out

merge-profiles:
	@rm -f coverage-all.out
	@gocovmerge *.out > coverage-all.out

test-coverage-func coverage-func: test-coverage merge-profiles
	@echo
	@echo "\033[1;34m=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-\033[0m"
	@echo "\033[1;34mFunctions NOT COVERED by Tests\033[0m"
	@echo "\033[1;34m=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-\033[0m"
	@go tool cover -func=coverage-all.out | egrep -v "100.0[%]"

mocks: agent-mock session-mock networkentity-mock pitaya-mock serializer-mock metrics-mock acceptor-mock cluster-mock

agent-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/agent Agent,AgentFactory | sed 's/mock_agent/mocks/' > pkg/agent/mocks/agent.go

session-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/session Session,SessionPool | sed 's/mock_session/mocks/' > pkg/session/mocks/session.go

networkentity-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/networkentity NetworkEntitgit y | sed 's/mock_networkentity/mocks/' > pkg/networkentity/mocks/networkentity.go

pitaya-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg Pitaya | sed 's/mock_v2/mocks/' > pkg/mocks/app.go

metrics-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/metrics Reporter | sed 's/mock_metrics/mocks/' > pkg/metrics/mocks/reporter.go
	@mockgen github.com/topfreegames/pitaya/v3/pkg/metrics Client | sed 's/mock_metrics/mocks/' > pkg/metrics/mocks/statsd_reporter.go

serializer-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/serialize Serializer | sed 's/mock_serialize/mocks/' > pkg/serialize/mocks/serializer.go

acceptor-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/acceptor PlayerConn,Acceptor | sed 's/mock_acceptor/mocks/' > pkg/mocks/acceptor.go

cluster-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/cluster RPCServer,RPCClient,SDListener,RemoteBindingListener,InfoRetriever | sed 's/mock_cluster/mocks/' > pkg/cluster/mocks/cluster.go
