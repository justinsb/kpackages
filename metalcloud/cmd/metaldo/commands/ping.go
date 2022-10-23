package commands

import (
	"context"
	"fmt"
	"log"

	agentv1 "github.com/justinsb/metalcloud/agent/v1"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/prototext"
)

type PingOptions struct {
	*RootOptions
}

func AddPingCommand(parent *cobra.Command, rootOptions *RootOptions) {
	var opt PingOptions
	opt.RootOptions = rootOptions

	cmd := &cobra.Command{
		Use: "ping",
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunPingCommand(cmd.Context(), opt)
	}

	parent.AddCommand(cmd)
}

func RunPingCommand(ctx context.Context, options PingOptions) error {
	agentService, err := Connect(ctx, options.RootOptions)
	if err != nil {
		return err
	}

	response, err := agentService.Ping(ctx, &agentv1.PingRequest{})
	if err != nil {
		return fmt.Errorf("failed to Ping: %w", err)
	} else {
		log.Printf("got ping response: %s\n", prototext.Format(response))
	}
	return nil

}
