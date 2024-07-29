ifeq ($(OS), Windows_NT)
	BIN := pitaya-cli.exe
	XK6_BIN := k6.exe
	MKFOLDER := if not exist "build" mkdir build
	GREP_CMD := findstr /V
else
	BIN := pitaya-cli
	XK6_BIN := k6
	MKFOLDER := mkdir -p build
	GREP_CMD := grep -v
endif

TESTABLE_PACKAGES = `go list ./... | $(GREP_CMD) examples | $(GREP_CMD) constants | $(GREP_CMD) mocks | $(GREP_CMD) helpers | $(GREP_CMD) interfaces | $(GREP_CMD) protos | $(GREP_CMD) e2e | $(GREP_CMD) benchmark`

setup: init-submodules
	@go get ./...

build:
	@$(MKFOLDER)
	@go build -o build/$(BIN) .
	@echo "build pitaya-cli at ./build/$(BIN)"

build-k6-extension:
	@$(MKFOLDER)
	@xk6 build --with github.com/topfreegames/xk6-pitaya=./xk6-pitaya/ --with github.com/topfreegames/pitaya/v3=./ --with google.golang.org/grpc=google.golang.org/grpc@v1.54.1 --output ./build/$(XK6_BIN)
	@echo "build pitaya k6 extension at ./build/$(XK6_BIN)"

init-submodules:
	@git submodule init

setup-ci:
	@go install github.com/mattn/goveralls@latest
	@go install github.com/wadey/gocovmerge@latest

setup-protobuf-macos:
	@brew install protobuf
	@go install github.com/golang/protobuf/protoc-gen-go@latest

run-jaeger-aio:
	@docker-compose -f ./examples/testing/docker-compose-jaeger.yml up -d
	@echo "Access jaeger UI @ http://localhost:16686"

run-chat-example:
	@cd examples/testing && docker-compose up -d etcd nats && cd ../demo/chat/ && go run main.go

run-cluster-example-frontend-tracing:
	@PITAYA_METRICS_PROMETHEUS_PORT=9090 OTEL_SDK_DISABLED=false OTEL_SERVICE_NAME=example-frontend OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 OTEL_EXPORTER_OTLP_PROTOCOL=grpc OTEL_TRACES_SAMPLER=parentbased_traceidratio OTEL_TRACES_SAMPLER_ARG="1" go run examples/demo/cluster/main.go

run-cluster-example-backend-tracing:
	@PITAYA_METRICS_PROMETHEUS_PORT=9091 OTEL_SDK_DISABLED=false OTEL_SERVICE_NAME=example-backend OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 OTEL_EXPORTER_OTLP_PROTOCOL=grpc OTEL_TRACES_SAMPLER=parentbased_traceidratio OTEL_TRACES_SAMPLER_ARG="1" go run examples/demo/cluster/main.go --port 3251 --type room --frontend=false

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
	@protoc -I pitaya-protos/ pitaya-protos/*.proto --go_out=plugins=grpc:protos
	@protoc -I pitaya-protos/test pitaya-protos/test/*.proto --go_out=protos/test

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

kill-jaeger:
	@docker-compose -f ./examples/testing/docker-compose-jaeger.yml down; true

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

mocks: agent-mock session-mock networkentity-mock pitaya-mock serializer-mock metrics-mock acceptor-mock

agent-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/agent Agent,AgentFactory | sed 's/mock_agent/mocks/' > pkg/agent/mocks/agent.go

session-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/session Session,SessionPool | sed 's/mock_session/mocks/' > pkg/session/mocks/session.go

networkentity-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/networkentity NetworkEntity | sed 's/mock_networkentity/mocks/' > pkg/networkentity/mocks/networkentity.go

pitaya-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg Pitaya | sed 's/mock_pkg/mocks/' > pkg/mocks/app.go

metrics-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/metrics Reporter | sed 's/mock_metrics/mocks/' > pkg/metrics/mocks/reporter.go
	@mockgen github.com/topfreegames/pitaya/v3/pkg/metrics Client | sed 's/mock_metrics/mocks/' > pkg/metrics/mocks/statsd_reporter.go

serializer-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/serialize Serializer | sed 's/mock_serialize/mocks/' > pkg/serialize/mocks/serializer.go

acceptor-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/acceptor PlayerConn,Acceptor | sed 's/mock_acceptor/mocks/' > pkg/mocks/acceptor.go

worker-mock:
	@mockgen github.com/topfreegames/pitaya/v3/pkg/worker RPCJob | sed 's/mock_worker/mocks/' > pkg/worker/mocks/rpc_job.go
