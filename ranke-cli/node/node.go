package node

import (
	"context"
	"fmt"

	"github.com/flocko-motion/ranke-cli/internal/cli"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node <id>",
		Short: "Show a node by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cli.Client()
			if err != nil {
				return err
			}
			ctx := context.Background()
			jsonOut, _ := cmd.Flags().GetBool("json")
			showEdges, _ := cmd.Flags().GetBool("edges")

			resp, err := c.GetApiNodesIdWithResponse(ctx, args[0])
			if err != nil {
				return err
			}
			if resp.JSON200 == nil {
				return fmt.Errorf("node not found (status %d)", resp.StatusCode())
			}

			if jsonOut {
				cli.PrintJSON(resp.JSON200)
			} else {
				cli.PrintNode(*resp.JSON200)
			}

			if showEdges {
				edgesResp, err := c.GetApiNodesIdEdgesWithResponse(ctx, args[0])
				if err != nil {
					return err
				}
				if edgesResp.JSON200 != nil && len(edgesResp.JSON200.Edges) > 0 {
					fmt.Printf("\nEdges:\n")
					cli.PrintEdgeTable(edgesResp.JSON200.Edges)
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	cmd.Flags().BoolP("edges", "e", false, "Also show connected edges")
	return cmd
}
