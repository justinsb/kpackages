package main

import (
	"context"
	"fmt"
	"log"
	"os"

	agentv1 "github.com/justinsb/metalcloud/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/prototext"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	connectTo := "10.78.78.32:8080"
	conn, err := grpc.Dial(connectTo, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to connect to service on %s: %w", connectTo, err)
	}
	log.Println("Connected to", connectTo)

	agentService := agentv1.NewAgentServiceClient(conn)

	args := os.Args[1:]
	if len(args) == 0 {
		return fmt.Errorf("expected: command [command args...]")
	}
	switch args[0] {
	case "ping":
		response, err := agentService.Ping(ctx, &agentv1.PingRequest{})
		if err != nil {
			return fmt.Errorf("failed to Ping: %w", err)
		} else {
			log.Printf("got ping response: %s\n", prototext.Format(response))
		}
	case "exec":
		req := &agentv1.ExecRequest{
			Args: args[1:],
		}
		response, err := agentService.Exec(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to Exec: %w", err)
		} else {
			log.Printf("got exec response: %s\n", prototext.Format(response))
		}
	// case "cp":
	// 	if len(args) != 3 {
	// 		return fmt.Errorf("syntax: cp <src> <dest>")
	// 	}
	// 	req := &agentv1.WriteFileRequest{
	// 		Dest: args[2],
	// 	}
	// 	response, err := agentService.Exec(ctx, req)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to Exec: %w", err)
	// 	} else {
	// 		log.Printf("got exec response: %s\n", prototext.Format(response))
	// 	}
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
	return nil
}
