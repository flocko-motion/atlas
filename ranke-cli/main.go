package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfg Config

func main() {
	rootCmd := &cobra.Command{
		Use:   "ranke-cli",
		Short: "CLI for interacting with a RankeDB server",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			cfg, err = loadConfig()
			if err != nil {
				return err
			}
			// Allow --server flag to override config
			if s, _ := cmd.Flags().GetString("server"); s != "" {
				cfg.Server = s
			}
			if cfg.Server == "" {
				return fmt.Errorf("no server configured — set server in ~/.rankedb/etc/ranke-cli.toml or use --server")
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().String("server", "", "RankeDB server URL (overrides config)")

	rootCmd.AddCommand(
		statusCmd(),
		nodesCmd(),
		nodeCmd(),
		entitiesCmd(),
		entityCmd(),
		relationsCmd(),
		edgeCmd(),
		provenanceCmd(),
		ingestCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
