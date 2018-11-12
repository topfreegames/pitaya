# pitaya-chat-demo
chat room demo base on [pitaya](https://github.com/topfreegames/pitaya) in 100 lines

refs: https://github.com/topfreegames/pitaya

## Required
- golang
- websocket

## Run
```
docker-compose -f ../../testing/docker-compose.yml up -d etcd nats
go run main.go
```

open browser => http://localhost:3251/web/
