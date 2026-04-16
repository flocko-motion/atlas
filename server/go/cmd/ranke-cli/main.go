package main

import (
	"fmt"
	"os"

	"rankedb/cmd/ranke-cli/edge"
	"rankedb/cmd/ranke-cli/entities"
	"rankedb/cmd/ranke-cli/entity"
	"rankedb/cmd/ranke-cli/ingest"
	"rankedb/cmd/ranke-cli/internal/cli"
	"rankedb/cmd/ranke-cli/node"
	"rankedb/cmd/ranke-cli/nodes"
	"rankedb/cmd/ranke-cli/provenance"
	"rankedb/cmd/ranke-cli/relations"
	"rankedb/cmd/ranke-cli/status"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ranke-cli",
		Short: "CLI for interacting with a RankeDB server",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := cli.LoadConfig(); err != nil {
				return err
			}
			if s, _ := cmd.Flags().GetString("server"); s != "" {
				cli.Cfg.Server = s
			}
			if cli.Cfg.Server == "" {
				return fmt.Errorf("no server configured — set server in %s or use --server", cli.ConfigPath())
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().String("server", "", "RankeDB server URL (overrides config)")

	rootCmd.AddCommand(
		status.Cmd(),
		nodes.Cmd(),
		node.Cmd(),
		entities.Cmd(),
		entity.Cmd(),
		relations.Cmd(),
		edge.Cmd(),
		provenance.Cmd(),
		ingest.Cmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
