# sandboxd — the sandbox execution sidecar (ADR-0002). Owns the Docker
# socket; serves mTLS gRPC on :8443. The app container never mounts the
# socket.
FROM golang:1.26.0-alpine3.23 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /sandboxd ./cmd/sandboxd

FROM alpine:3.21
RUN apk add --no-cache docker-cli
COPY --from=build /sandboxd /usr/local/bin/sandboxd
# Runs as root: the sidecar's sole job is owning the docker socket (its
# compromise == host docker anyway, and the untrusted code runs inside
# throwaway containers launched by it).
EXPOSE 8443
ENTRYPOINT ["sandboxd"]
