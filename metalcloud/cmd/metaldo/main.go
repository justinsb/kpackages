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
	rootOptions := commands.RootOptions{}
	rootOptions.InitDefaults()

	root := &cobra.Command{
		Use: "metaldo",
	}
	root.SilenceUsage = true
	root.PersistentFlags().StringVar(&rootOptions.Host, "host", rootOptions.Host, "host to connect to")

	commands.AddExecCommand(root, &rootOptions)
	commands.AddPingCommand(root, &rootOptions)

	return root.ExecuteContext(ctx)
}
