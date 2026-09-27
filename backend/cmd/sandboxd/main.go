// sandboxd — the sandbox execution sidecar (ADR-0002). Owns the Docker
// socket; serves the mTLS gRPC Execute RPC on an internal port.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/intivai/backend/internal/sandbox/proto"
	"github.com/intivai/backend/internal/sandbox/sidecar"
	"github.com/intivai/backend/pkg/telemetry"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	if err := run(); err != nil {
		log.Error().Err(err).Msg("sandboxd fatal error")
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Tracing (otel-tracing-plan-2026-08-26): spans arrive with Batch F
	// (grpc interceptors); bootstrap now so the service name exists.
	shutdownTracing, err := telemetry.Init(ctx, telemetry.Config{
		Enable:       os.Getenv("INTIVAI_OTEL_ENABLE") == "true",
		ServiceName:  "intivai-sandboxd",
		Env:          envOr("INTIVAI_ENV", "dev"),
		OTLPEndpoint: envOr("INTIVAI_OTEL_EXPORTER_OTLP_ENDPOINT", "http://jaeger:4318"),
		SampleRatio:  sampleRatio(),
	})
	if err != nil {
		return fmt.Errorf("telemetry: %w", err)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	addr := envOr("SANDBOXD_ADDR", ":8443")
	caFile := os.Getenv("SANDBOXD_CA")
	certFile := os.Getenv("SANDBOXD_CERT")
	keyFile := os.Getenv("SANDBOXD_KEY")
	if caFile == "" || certFile == "" || keyFile == "" {
		return errors.New("SANDBOXD_CA / SANDBOXD_CERT / SANDBOXD_KEY required (mTLS)")
	}

	tlsConfig, err := loadServerTLS(caFile, certFile, keyFile)
	if err != nil {
		return fmt.Errorf("load mTLS material: %w", err)
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srv := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	proto.RegisterSandboxServiceServer(srv, sidecar.NewGRPCServer(sidecar.NewRunner()))

	go func() {
		<-ctx.Done()
		log.Info().Msg("shutting down")
		srv.GracefulStop()
	}()

	log.Info().Str("addr", addr).Msg("sandboxd serving mTLS gRPC")
	if err := srv.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

func loadServerTLS(caFile, certFile, keyFile string) (*tls.Config, error) {
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("invalid CA pem")
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		// The app is the only caller — require its client cert.
		ClientAuth: tls.RequireAndVerifyClientCert,
		MinVersion: tls.VersionTLS13,
	}, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func sampleRatio() float64 {
	r, err := strconv.ParseFloat(envOr("INTIVAI_OTEL_TRACES_SAMPLER_ARG", "1"), 64)
	if err != nil || r <= 0 || r > 1 {
		return 1
	}
	return r
}
