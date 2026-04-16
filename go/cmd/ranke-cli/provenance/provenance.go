package provenance

import (
	"context"
	"fmt"

	"github.com/flocko-motion/rankedb/go/cmd/ranke-cli/internal/cli"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provenance <node-id>",
		Short: "Show the provenance chain of a node back to L0 roots",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cli.Client()
			if err != nil {
				return err
			}
			ctx := context.Background()
			jsonOut, _ := cmd.Flags().GetBool("json")

			resp, err := c.GetApiNodesIdProvenanceWithResponse(ctx, args[0])
			if err != nil {
				return err
			}
			if resp.JSON200 == nil {
				return fmt.Errorf("node not found (status %d)", resp.StatusCode())
			}

			if jsonOut {
				cli.PrintJSON(resp.JSON200)
				return nil
			}

			fmt.Printf("Provenance chain for %s\n\n", args[0])
			fmt.Printf("Nodes (%d):\n", len(resp.JSON200.Nodes))
			cli.PrintNodeTable(resp.JSON200.Nodes)
			fmt.Printf("\nEdges (%d):\n", len(resp.JSON200.Edges))
			cli.PrintEdgeTable(resp.JSON200.Edges)
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}
