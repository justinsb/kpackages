package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	agentv1 "github.com/justinsb/metalcloud/agent/v1"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/prototext"
)

type ExecOptions struct {
	*RootOptions

	Chroot  string
	Replace bool
	Args    []string

	Stdin chan []byte
}

func AddExecCommand(parent *cobra.Command, rootOptions *RootOptions) {
	var opt ExecOptions
	opt.RootOptions = rootOptions

	cmd := &cobra.Command{
		Use: "exec",
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opt.Args = args
		opt.Stdin = make(chan []byte)
		go func() {
			buf := make([]byte, 4096)
			for {
				if n, err := os.Stdin.Read(buf); err != nil {
					close(opt.Stdin)
					log.Printf("error reading from stdin (n=%d): %v", n, err)
					return
				} else if n != 0 {
					c := make([]byte, n)
					copy(c, buf[:n])
					opt.Stdin <- c
				}
			}
		}()
		return RunExecCommand(cmd.Context(), opt)
	}

	cmd.Flags().StringVar(&opt.Chroot, "chroot", opt.Chroot, "run in chroot")
	cmd.Flags().BoolVar(&opt.Replace, "replace", opt.Replace, "replace the running process... DANGEROUS")

	parent.AddCommand(cmd)
}

func RunExecCommand(ctx context.Context, opt ExecOptions) error {
	agentService, err := Connect(ctx, opt.RootOptions)
	if err != nil {
		return err
	}

	exec := &agentv1.CommandExecution{
		Args:   opt.Args,
		Chroot: opt.Chroot,
	}

	if opt.Replace {
		exec.Replace = true
		req := &agentv1.ExecRequest{
			Exec: exec,
		}
		response, err := agentService.Exec(ctx, req)
		// TODO: Not expected to return
		if err != nil {
			return fmt.Errorf("failed to Exec: %w", err)
		} else {
			log.Printf("got exec response: %s\n", prototext.Format(response))
			os.Stdout.Write(response.Stdout)
		}
		return nil
	}

	stream, err := agentService.ExecStreaming(ctx)
	if err != nil {
		return fmt.Errorf("failed to ExecStreaming: %w", err)
	}

	{
		req := &agentv1.ExecStreamingRequest{
			Exec: exec,
		}
		if err := stream.Send(req); err != nil {
			return err
		}
	}

	forwardStdinCtx, cancelStdin := context.WithCancel(ctx)

	// stdin := os.Stdin
	// pr, pw := io.Pipe()
	errorChan := make(chan error, 2)
	go func() {
		// buf := make([]byte, 4096)
		for {
			select {
			case b, ok := <-opt.Stdin:
				if !ok {

					req := &agentv1.ExecStreamingRequest{
						CloseStdin: true,
					}
					if err := stream.Send(req); err != nil {
						errorChan <- err
						return
					}
					errorChan <- nil
					return
				}
				if len(b) != 0 {
					req := &agentv1.ExecStreamingRequest{
						Stdin: b,
					}
					if err := stream.Send(req); err != nil {
						errorChan <- err
						return
					}
				}
			case <-forwardStdinCtx.Done():
				errorChan <- nil
				return
			}
		}
	}()

	go func() {
		defer func() {
			cancelStdin()
		}()

		for {
			msg, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					errorChan <- nil
					return
				}
				errorChan <- fmt.Errorf("error reading from grpc stream: %w", err)
				return
			}
			if len(msg.Stdout) != 0 {
				os.Stdout.Write(msg.Stdout)
			}
			if len(msg.Stderr) != 0 {
				os.Stderr.Write(msg.Stderr)
			}
			if msg.ExitInfo != nil {
				exitInfo := msg.GetExitInfo()
				log.Printf("exited with code %v", exitInfo.ExitCode)
				if err := stream.CloseSend(); err != nil {
					errorChan <- fmt.Errorf("error closing grpc stream: %w", err)
					return
				}
				// errorChan <- nil
				// return
			}
		}
	}()

	for i := 0; i < 2; i++ {
		err := <-errorChan
		if err != nil {
			return err
		}
	}
	return nil
}
