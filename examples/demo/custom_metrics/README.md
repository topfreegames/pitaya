Custom Metrics Example
=======================

# Run
`make run-custom-metrics-example`

# Call routes
By using [pitaya-cli](https://github.com/topfreegames/pitaya-cli), call:
```
connect localhost:3250
request room.room.setcounter {"value": 1.0, "tag1": "value1", "tag2": "value2"}
request room.room.setgauge1 {"value": 1.0, "tag1": "value1"}
request room.room.setgauge2 {"value": 1.0, "tag2": "value2"}
request room.room.setsummary {"value": 1.0, "tag1": "value1"}
```

Check out the results on `curl localhost:9090/metrics | grep 'my_'`
