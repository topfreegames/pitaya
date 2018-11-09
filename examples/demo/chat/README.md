# pitaya-chat-demo
chat room demo base on [pitaya](https://github.com/topfreegames/pitaya) in 100 lines

refs: https://github.com/topfreegames/pitaya

## Required
- golang
- websocket

## Run
```
cd ../../testing
docker-compose up -d etcd nats
cd ../demo/chat/
go run main.go
```

open browser => http://localhost:3251/web/
