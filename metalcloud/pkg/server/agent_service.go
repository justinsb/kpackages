package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"

	pb "github.com/justinsb/metalcloud/agent/v1"
	"golang.org/x/sys/unix"
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

// Exec serves the ping service method
func (s *agentService) Exec(ctx context.Context, req *pb.ExecRequest) (*pb.ExecResponse, error) {
	args := req.GetExec().GetArgs()
	if len(args) == 0 {
		return nil, fmt.Errorf("field args is required")
	}

	if req.GetExec().GetReplace() {
		var envv []string
		if err := unix.Exec(args[0], args, envv); err != nil {
			return nil, err
		}
		// Not expected to return
		return nil, fmt.Errorf("unexpected return from command with replace")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if len(req.Stdin) != 0 {
		cmd.Stdin = bytes.NewReader(req.Stdin)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Chroot = req.GetExec().GetChroot()

	response := &pb.ExecResponse{}
	response.ExitInfo = &pb.ExitInfo{}
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			response.ExitInfo.ExitCode = int32(exitError.ExitCode())
		} else {
			// TODO: return as valid response?
			return nil, err
		}
	}

	response.Stdout = stdout.Bytes()
	response.Stderr = stderr.Bytes()

	return response, nil
}

// Exec serves the ping service method
func (s *agentService) ExecStreaming(stream pb.AgentService_ExecStreamingServer) error {
	ctx := stream.Context()

	req, err := stream.Recv()
	if err != nil {
		return err
	}

	args := req.GetExec().GetArgs()
	if len(args) == 0 {
		return fmt.Errorf("field args is required")
	}

	// TODO: Deduplciate
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Chroot = req.GetExec().GetChroot()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stderr pipe: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("error creating stdin pipe: %w", err)
	}
	defer stdin.Close()

	errorChan := make(chan error, 4)
	go func() {
		for {
			buf := make([]byte, 4096)
			n, err := stderr.Read(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					errorChan <- nil
					log.Printf("stderr done")
					return
				} else if errors.Is(err, os.ErrClosed) {
					errorChan <- nil
					log.Printf("stderr done")
					return
				} else {
					errorChan <- fmt.Errorf("error reading from stderr: %w", err)
					return
				}
			}
			response := &pb.ExecStreamingResponse{}
			response.Stderr = buf[:n]
			if err := stream.Send(response); err != nil {
				errorChan <- fmt.Errorf("error sending to grpc stream: %w", err)
				return
			}
		}
	}()
	go func() {
		for {
			buf := make([]byte, 4096)
			n, err := stdout.Read(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					errorChan <- nil
					log.Printf("stdin done")
					return
				} else if errors.Is(err, os.ErrClosed) {
					errorChan <- nil
					log.Printf("stdin done")
					return
				} else {
					errorChan <- fmt.Errorf("error reading from stdout: %w", err)
					return
				}
			}
			response := &pb.ExecStreamingResponse{}
			response.Stdout = buf[:n]
			if err := stream.Send(response); err != nil {
				errorChan <- fmt.Errorf("error sending to grpc stream: %w", err)
				return
			}
		}
	}()

	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					errorChan <- nil
					return
				}
				errorChan <- fmt.Errorf("error reading from grpc stream: %w", err)
				return
			}
			if len(req.GetExec().GetArgs()) != 0 {
				errorChan <- fmt.Errorf("unexpected command-start request after first request")
				return
			}
			if len(req.Stdin) != 0 {
				if _, err := stdin.Write(req.Stdin); err != nil {
					errorChan <- fmt.Errorf("error writing to stdin: %w", err)
					return
				}
			}
			if req.CloseStdin {
				if err := stdin.Close(); err != nil {
					errorChan <- fmt.Errorf("error closing stdin: %w", err)
					return
				} else {
					errorChan <- nil
				}
			}
		}
	}()

	go func() {
		response := &pb.ExecStreamingResponse{}
		response.ExitInfo = &pb.ExitInfo{}

		if err := cmd.Run(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				response.ExitInfo.ExitCode = int32(exitError.ExitCode())
			} else {
				// TODO: return as valid response?
				errorChan <- fmt.Errorf("error from running command: %w", err)
				return
			}
		}

		if err := stream.Send(response); err != nil {
			errorChan <- fmt.Errorf("error writing exitcode to stream: %w", err)
			return
		}

		log.Printf("run done")
		errorChan <- nil
	}()

	for i := 0; i < 4; i++ {
		err := <-errorChan
		if err != nil {
			log.Printf("got error from goroutine: %v", err)

			response := &pb.ExecStreamingResponse{}
			response.Stderr = []byte(fmt.Sprintf("error:%v\n", err))
			stream.Send(response)

			return err
		}
		log.Printf("count %d", i)
	}
	return nil
}
