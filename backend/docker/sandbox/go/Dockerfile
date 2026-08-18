# Sandbox execution image: Go (ADR-0002). Code arrives on stdin; /tmp is the
# writable tmpfs; the root fs is read-only when launched.
FROM golang:1.26-alpine
RUN adduser -D -u 10001 sandbox
USER sandbox
ENV GOCACHE=/tmp/gocache GOPATH=/tmp/gopath GOMEMLIMIT=256MiB
WORKDIR /tmp
