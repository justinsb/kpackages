package server

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	pb "github.com/justinsb/metalcloud/agent/v1"
)

func NewAgentService() *agentService {
	return &agentService{}
}

// agentService implements the Agent GRPC Service.
type agentService struct {
	pb.UnimplementedAgentServiceServer
}

// Ping serves the ping service method
func (s *agentService) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{}, nil
}

// Ping serves the ping service method
func (s *agentService) Exec(ctx context.Context, req *pb.ExecRequest) (*pb.ExecResponse, error) {
	args := req.Args
	if len(args) == 0 {
		return nil, fmt.Errorf("field args is required")
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	response := &pb.ExecResponse{}
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			response.ExitCode = int32(exitError.ExitCode())
		} else {
			// TODO: return as valid response?
			return nil, err
		}
	}

	response.Stdout = stdout.Bytes()
	response.Stderr = stderr.Bytes()

	return response, nil
}
