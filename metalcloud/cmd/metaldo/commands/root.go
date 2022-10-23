package commands

import (
	"context"
	"fmt"
	"log"

	agentv1 "github.com/justinsb/metalcloud/agent/v1"
	"google.golang.org/grpc"
)

type RootOptions struct {
	Server string
}

func Connect(ctx context.Context, options *RootOptions) (agentv1.AgentServiceClient, error) {
	connectTo := options.Server

	conn, err := grpc.Dial(connectTo, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service on %s: %w", connectTo, err)
	}
	log.Println("Connected to", connectTo)

	agentService := agentv1.NewAgentServiceClient(conn)
	return agentService, nil
}
