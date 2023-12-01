package commands

import (
	"context"
	"fmt"
	"log"
	"os"

	agentv1 "github.com/justinsb/metalcloud/agent/v1"
	"google.golang.org/grpc"
)

type RootOptions struct {
	Host string
}

func (o *RootOptions) InitDefaults() {
	o.Host = os.Getenv("METAL_HOST")
}

func Connect(ctx context.Context, options *RootOptions) (agentv1.AgentServiceClient, error) {
	connectTo := options.Host
	if connectTo == "" {
		return nil, fmt.Errorf("must specify host")
	}
	connectTo += ":8080"
	log.Printf("connecting to %q", options.Host)
	conn, err := grpc.Dial(connectTo, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service on %s: %w", connectTo, err)
	}
	log.Println("Connected to", connectTo)

	agentService := agentv1.NewAgentServiceClient(conn)
	return agentService, nil
}
