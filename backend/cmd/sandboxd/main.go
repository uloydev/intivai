// sandboxd — the sandbox execution sidecar (ADR-0002). Owns the Docker
// socket; serves the mTLS gRPC Execute RPC on an internal port.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/intivai/backend/internal/sandbox/proto"
	"github.com/intivai/backend/internal/sandbox/sidecar"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := envOr("SANDBOXD_ADDR", ":8443")
	caFile := os.Getenv("SANDBOXD_CA")
	certFile := os.Getenv("SANDBOXD_CERT")
	keyFile := os.Getenv("SANDBOXD_KEY")
	if caFile == "" || certFile == "" || keyFile == "" {
		log.Fatal().Msg("SANDBOXD_CA / SANDBOXD_CERT / SANDBOXD_KEY required (mTLS)")
	}

	tlsConfig, err := loadServerTLS(caFile, certFile, keyFile)
	if err != nil {
		log.Fatal().Err(err).Msg("load mTLS material")
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal().Err(err).Msg("listen")
	}

	srv := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	proto.RegisterSandboxServiceServer(srv, sidecar.NewGRPCServer(sidecar.NewRunner()))

	go func() {
		<-ctx.Done()
		log.Info().Msg("shutting down")
		srv.GracefulStop()
	}()

	log.Info().Str("addr", addr).Msg("sandboxd serving mTLS gRPC")
	if err := srv.Serve(lis); err != nil {
		log.Fatal().Err(err).Msg("serve")
	}
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
