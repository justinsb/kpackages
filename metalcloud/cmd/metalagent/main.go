package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	agentv1 "github.com/justinsb/metalcloud/agent/v1"
	"github.com/justinsb/metalcloud/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	listen := "0.0.0.0:8080"
	clientCertificatePath := ""
	serverKeyPath := ""
	serverCertPath := ""
	flag.StringVar(&listen, "listen", listen, "endpoint on which to listen")
	flag.StringVar(&clientCertificatePath, "client-ca", clientCertificatePath, "path to client CA file")
	flag.StringVar(&serverCertPath, "server-cert", serverCertPath, "path to server cert file")
	flag.StringVar(&serverKeyPath, "server-key", serverKeyPath, "path to server key file")
	flag.Parse()

	listener, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", listen, err)
	}

	var opts []grpc.ServerOption

	cert, err := tls.LoadX509KeyPair(serverCertPath, serverKeyPath)
	if err != nil {
		return fmt.Errorf("loading TLS keypair: %w", err)
	}

	caPool := x509.NewCertPool()
	caBytes, err := os.ReadFile(clientCertificatePath)
	if err != nil {
		return fmt.Errorf("reading client certificate CA file %q: %w", clientCertificatePath, err)
	}
	if ok := caPool.AppendCertsFromPEM(caBytes); !ok {
		return fmt.Errorf("parsing client certificate CA file %q: %w", clientCertificatePath, err)
	}

	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
	}
	opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))

	grpcServer := grpc.NewServer(opts...)

	agentv1.RegisterAgentServiceServer(grpcServer, server.NewAgentService())
	log.Println("Listening on", listen)
	if err := grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %w", err)
	}

	return nil
}
