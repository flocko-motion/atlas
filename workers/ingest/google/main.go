package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ranke-ingest-google <path-to-takeout-dir>",
		Short: "Ingest a Google Takeout export into RankeDB",
		Long: `Ingests Google Takeout data into RankeDB as L0 source nodes.

Auto-detects the takeout type from directory contents:
  - Contacts (*.vcf files + *.jpg photos)
  - Calendar (*.ics files)  [planned]

Each VCF entry becomes a source/data node (encoding: text/vcard).
Each contact photo becomes a source/media node (encoding: image/jpeg).
Group membership is preserved in the VCF CATEGORIES field.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			server, _ := cmd.Flags().GetString("server")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			return run(args[0], server, dryRun)
		},
	}

	rootCmd.Flags().String("server", "http://localhost:7000", "RankeDB server URL")
	rootCmd.Flags().Bool("dry-run", false, "Parse and report without sending to server")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
