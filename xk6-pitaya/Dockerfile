# k6 release to build. Keep it in sync with go.k6.io/k6 in xk6-pitaya/go.mod:
# the latest xk6 builds k6 v2 by default, which does not load extensions
# written against go.k6.io/k6 v1.
ARG K6_VERSION=1.4.1

# Build the k6 binary with the extension
FROM golang:1.27 AS builder

ARG pitaya_revision
ARG K6_VERSION

RUN go install go.k6.io/xk6/cmd/xk6@latest
# For our example, we'll add support for output of test metrics to InfluxDB v2.
# Feel free to add other extensions using the '--with ...'.
RUN xk6 build v${K6_VERSION} \
    --with github.com/topfreegames/pitaya/xk6-pitaya@$pitaya_revision  \
    --replace github.com/topfreegames/pitaya/v3=github.com/topfreegames/pitaya/v3@$pitaya_revision \
    --output /k6

# Use the operator's base image and override the k6 binary
FROM grafana/k6:${K6_VERSION}
COPY --from=builder /k6 /usr/bin/k6
