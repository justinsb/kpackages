package main

import (
	"context"
	"fmt"
	"os"

	"github.com/justinsb/metalcloud/cmd/metaldo/commands"
	"github.com/spf13/cobra"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	rootOptions := commands.RootOptions{
		Server: "192.168.76.9:8080",
	}
	root := &cobra.Command{
		Use: "metaldo",
	}
	root.SilenceUsage = true

	commands.AddExecCommand(root, &rootOptions)
	commands.AddPingCommand(root, &rootOptions)

	return root.ExecuteContext(ctx)
}
