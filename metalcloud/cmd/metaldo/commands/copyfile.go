package commands

import (
	"context"
	"fmt"
	"log"
	"os"

	agentv1 "github.com/justinsb/metalcloud/agent/v1"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/prototext"
)

type CopyFileOptions struct {
	*RootOptions
}

func AddCopyFileCommand(parent *cobra.Command, rootOptions *RootOptions) {
	var opt CopyFileOptions
	opt.RootOptions = rootOptions

	cmd := &cobra.Command{
		Use: "cp",
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCopyFileCommand(cmd.Context(), opt, args)
	}

	parent.AddCommand(cmd)
}

func RunCopyFileCommand(ctx context.Context, opt CopyFileOptions, args []string) error {
	agentService, err := Connect(ctx, opt.RootOptions)
	if err != nil {
		return err
	}

	if len(args) != 2 {
		return fmt.Errorf("expected exactly two arguments: <src> <dest>")
	}

	b, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("reading file %q: %w", args[0], err)
	}
	req := &agentv1.WriteFileRequest{
		Path:     args[1],
		Contents: b,
		FileMode: 0644,
	}

	response, err := agentService.WriteFile(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to WriteFile: %w", err)
	} else {
		log.Printf("got copy response: %s\n", prototext.Format(response))
	}
	return nil
}
