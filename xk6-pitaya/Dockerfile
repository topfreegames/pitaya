# Build the k6 binary with the extension
FROM golang:1.20 as builder

ARG pitaya_revision

RUN go install go.k6.io/xk6/cmd/xk6@latest
# For our example, we'll add support for output of test metrics to InfluxDB v2.
# Feel free to add other extensions using the '--with ...'.
RUN xk6 build \
    --with github.com/topfreegames/pitaya/xk6-pitaya@$pitaya_revision  \
    --with github.com/topfreegames/pitaya/v3@$pitaya_revision \
    --output /k6

# Use the operator's base image and override the k6 binary
FROM grafana/k6:latest
COPY --from=builder /k6 /usr/bin/k6
