package main

import (
	"fmt"
	"os"

	"github.com/flocko-motion/ranke-cli/edge"
	"github.com/flocko-motion/ranke-cli/entities"
	"github.com/flocko-motion/ranke-cli/entity"
	"github.com/flocko-motion/ranke-cli/ingest"
	"github.com/flocko-motion/ranke-cli/internal/cli"
	"github.com/flocko-motion/ranke-cli/node"
	"github.com/flocko-motion/ranke-cli/nodes"
	"github.com/flocko-motion/ranke-cli/provenance"
	"github.com/flocko-motion/ranke-cli/relations"
	"github.com/flocko-motion/ranke-cli/status"
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
