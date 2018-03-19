setup:
	@dep ensure

run-chat-example:
	@go run examples/demo/chat/main.go

run-cluster-example:
	@go run examples/demo/cluster/main.go

run-tadpole-example:
	@go run examples/demo/tadpole/main.go
