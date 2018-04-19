TESTABLE_PACKAGES = `go list ./... | grep -v examples | grep -v constants | grep -v mocks | grep -v helpers | grep -v interfaces | grep -v protos | grep -v e2e | grep -v benchmark`

setup:
	@dep ensure

setup-ci:
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
	@go run examples/demo/cluster/main.go

run-cluster-protobuf-frontend-example:
	@cd examples/demo/cluster_protobuf && go run main.go

run-cluster-protobuf-backend-example:
	@cd examples/demo/cluster_protobuf && go run main.go --port 3251 --type room --frontend=false

run-cluster-example-backend:
	@go run examples/demo/cluster/main.go --port 3251 --type room --frontend=false

run-tadpole-example:
	@go run examples/demo/tadpole/main.go

protos-compile:
	@cd benchmark/testdata && ./gen_proto.sh
	@cd protos && protoc --gogofaster_out=. pitaya.proto

rm-test-temp-files:
	@rm -f cluster/127.0.0.1* 127.0.0.1*
	@rm -f cluster/localhost* localhost*

ensure-e2e-bin:
	@[ -f ./e2e/server/server ] || go build -o ./e2e/server/server ./e2e/server/main.go

ensure-e2e-deps:
	@cd ./e2e/server && docker-compose up -d

kill-e2e-deps:
	@cd ./e2e/server && docker-compose down; true

e2e-test: ensure-e2e-deps ensure-e2e-bin
	@echo "===============RUNNING E2E TESTS==============="
	@go test ./e2e/e2e_test.go

e2e-test-update: ensure-e2e-deps ensure-e2e-bin
	@echo "===============RUNNING E2E TESTS==============="
	@go test ./e2e/e2e_test.go -update

unit-test-coverage: kill-e2e-deps
	@echo "===============RUNNING UNIT TESTS==============="
	@go test $(TESTABLE_PACKAGES) -coverprofile coverprofile.out

test: kill-e2e-deps test-coverage
	@make rm-test-temp-files
	@make ensure-e2e-deps
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
