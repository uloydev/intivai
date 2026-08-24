# sandboxd — the sandbox execution sidecar (ADR-0002). Owns the Docker
# socket; serves mTLS gRPC on :8443. The app container never mounts the
# socket.
FROM golang:1.26.1-alpine3.23@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /sandboxd ./cmd/sandboxd

# 3.23 matches the builder minor (was 3.21); digest-pinned (A14)
FROM alpine:3.23@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40
RUN apk add --no-cache docker-cli
COPY --from=build /sandboxd /usr/local/bin/sandboxd
# Runs as root: the sidecar's sole job is owning the docker socket (its
# compromise == host docker anyway, and the untrusted code runs inside
# throwaway containers launched by it).
EXPOSE 8443
ENTRYPOINT ["sandboxd"]
