package commands

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"

	agentv1 "github.com/justinsb/metalcloud/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"k8s.io/klog/v2"
)

type RootOptions struct {
	Name           string
	Host           string
	ClientCertPath string
	ClientKeyPath  string
	ServerCAPath   string
}

func (o *RootOptions) InitDefaults() {
	o.Host = os.Getenv("METAL_HOST")
}

func Connect(ctx context.Context, options *RootOptions) (agentv1.AgentServiceClient, error) {
	connectTo := options.Host
	if connectTo == "" {
		return nil, fmt.Errorf("must specify host")
	}

	var opts []grpc.DialOption

	// opts = append(opts, grpc.WithBlock())
	if options.ClientCertPath != "" {
		cert, err := tls.LoadX509KeyPair(options.ClientCertPath, options.ClientKeyPath)
		if err != nil {
			log.Fatalf("failed to load client cert: %v", err)
		}

		ca := x509.NewCertPool()
		caFilePath := options.ServerCAPath
		caBytes, err := os.ReadFile(caFilePath)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", caFilePath, err)
		}
		if ok := ca.AppendCertsFromPEM(caBytes); !ok {
			return nil, fmt.Errorf("unable to parse certificates from %q", caFilePath)
		}

		tlsConfig := &tls.Config{
			ServerName:   options.Name,
			Certificates: []tls.Certificate{cert},
			RootCAs:      ca,
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))

		opts = append(opts, grpc.WithContextDialer(func(ctx context.Context, name string) (net.Conn, error) {
			dialer := &net.Dialer{}
			return dialer.DialContext(ctx, "tcp", options.Host+":8443")
		}))
		connectTo = options.Name + ":8443"
	} else {
		klog.Warningf("using insecure connection")
		connectTo += ":8080"
		opts = append(opts, grpc.WithInsecure())
	}

	log.Printf("connecting to %q", connectTo)
	conn, err := grpc.Dial(connectTo, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service on %s: %w", connectTo, err)
	}
	log.Println("Connected to", connectTo)

	agentService := agentv1.NewAgentServiceClient(conn)
	return agentService, nil
}
