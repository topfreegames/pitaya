TESTABLE_PACKAGES = `go list ./... | grep -v examples | grep -v constants | grep -v mocks | grep -v helpers | grep -v interfaces | grep -v protos | grep -v e2e | grep -v benchmark`

setup: init-submodules protos-compile
	@dep ensure

init-submodules:
	@git submodule init

setup-ci: init-submodules protos-compile
	@go get github.com/mattn/goveralls
	@go get -u github.com/golang/dep/cmd/dep
	@go get -u github.com/wadey/gocovmerge
	@dep ensure

setup-protobuf-macos:
	@brew install protobuf
	@go get -u github.com/gogo/protobuf/protoc-gen-gogofaster

run-chat-example:
	@cd examples/demo/chat/ && go run main.go

run-cluster-example-frontend:
	@PITAYA_METRICS_PROMETHEUS_PORT=9090 go run examples/demo/cluster/main.go

run-cluster-protobuf-frontend-example:
	@cd examples/demo/cluster_protobuf && go run main.go

run-cluster-protobuf-backend-example:
	@cd examples/demo/cluster_protobuf && go run main.go --port 3251 --type room --frontend=false

run-cluster-example-backend:
	@PITAYA_METRICS_PROMETHEUS_PORT=9091 go run examples/demo/cluster/main.go --port 3251 --type room --frontend=false

run-tadpole-example:
	@go run examples/demo/tadpole/main.go

protos-compile:
	@cd benchmark/testdata && ./gen_proto.sh
	@protoc -I protos/ protos/*.proto --gogofaster_out=plugins=grpc:protos
	@protoc -I protos/test protos/test/*.proto --gogofaster_out=protos/test

rm-test-temp-files:
	@rm -f cluster/127.0.0.1* 127.0.0.1*
	@rm -f cluster/localhost* localhost*

ensure-testing-bin:
	@[ -f ./examples/testing/server ] || go build -o ./examples/testing/server ./examples/testing/main.go

ensure-testing-deps:
	@cd ./examples/testing && docker-compose up -d

kill-testing-deps:
	@cd ./examples/testing && docker-compose down; true

e2e-test: e2e-test-nats e2e-test-grpc

e2e-test-nats: ensure-testing-deps ensure-testing-bin
	@echo "===============RUNNING E2E NATS TESTS==============="
	@go test ./e2e/e2e_test.go -update

e2e-test-grpc: ensure-testing-deps ensure-testing-bin
	@echo "===============RUNNING E2E GRPC TESTS==============="
	@go test ./e2e/e2e_test.go -update -grpc

benchmark-test: ensure-testing-deps ensure-testing-bin
	@echo "===============RUNNING BENCHMARK TESTS==============="
	@echo "--- starting testing servers"
	@PITAYA_METRICS_PROMETHEUS_PORT=9098 ./examples/testing/server -type game -frontend=false > /dev/null 2>&1 & echo $$! > back.PID
	@PITAYA_METRICS_PROMETHEUS_PORT=9099 ./examples/testing/server -type connector -frontend=true > /dev/null 2>&1 & echo $$! > front.PID
	@echo "--- sleeping for 5 seconds"
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
