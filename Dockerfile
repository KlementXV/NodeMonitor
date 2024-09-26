FROM golang:1.23.1-alpine3.20 AS builder

COPY main.go go.mod go.sum .

RUN go build -o NodeMonitor main.go

FROM alpine:3.20.3

RUN addgroup -S monitor && adduser -S monitor -G monitor

COPY --from=builder --chown=monitor:monitor --chmod=500 /go/NodeMonitor /usr/local/bin/NodeMonitor

USER monitor

ENTRYPOINT ["NodeMonitor"]