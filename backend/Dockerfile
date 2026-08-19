# Build
FROM golang:1.26.0-alpine3.20 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 go build -o /out/sandboxd ./cmd/sandboxd

# Run
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata tesseract-ocr tesseract-ocr-data-eng poppler-utils ghostscript
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /out/server .
COPY --from=builder --chown=appuser:appgroup /out/sandboxd .
USER appuser
EXPOSE 8080
ENTRYPOINT ["./server"]
