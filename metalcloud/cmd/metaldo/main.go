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
	root.PersistentFlags().StringVar(&rootOptions.Name, "name", rootOptions.Name, "expected name of server (in certificate)")
	root.PersistentFlags().StringVar(&rootOptions.ClientCertPath, "client-cert", rootOptions.ClientCertPath, "path to client certificate")
	root.PersistentFlags().StringVar(&rootOptions.ClientKeyPath, "client-key", rootOptions.ClientKeyPath, "path to client key")
	root.PersistentFlags().StringVar(&rootOptions.ServerCAPath, "server-ca", rootOptions.ServerCAPath, "path to server CA")

	commands.AddCopyFileCommand(root, &rootOptions)
	commands.AddExecCommand(root, &rootOptions)
	commands.AddPingCommand(root, &rootOptions)

	return root.ExecuteContext(ctx)
}
