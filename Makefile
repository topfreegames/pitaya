setup:
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
