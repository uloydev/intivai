# Build (digest-pinned: mutable tags rebuilt under us — A14)
FROM golang:1.26.1-alpine3.23@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 go build -o /out/sandboxd ./cmd/sandboxd

# Run (3.23 matches the builder minor; was 3.20 — A14)
FROM alpine:3.23@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40
RUN apk add --no-cache ca-certificates tzdata tesseract-ocr tesseract-ocr-data-eng poppler-utils ghostscript
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
# Read-only-rootfs compatible: /data is the app's only writable persistent
# path (memory banks, embedding models); /tmp is provided as tmpfs at runtime.
RUN mkdir -p /data && chown appuser:appgroup /data
VOLUME /data
WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /out/server .
COPY --from=builder --chown=appuser:appgroup /out/sandboxd .
USER appuser
EXPOSE 8080
ENTRYPOINT ["./server"]
