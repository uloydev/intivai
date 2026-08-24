# Sandbox execution image: Go (ADR-0002). Code arrives on stdin; /tmp is the
# writable tmpfs; the root fs is read-only when launched.
FROM golang:1.26.1-alpine3.23@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039
RUN adduser -D -u 10001 sandbox
USER sandbox
ENV GOCACHE=/tmp/gocache GOPATH=/tmp/gopath GOMEMLIMIT=256MiB
WORKDIR /tmp
