package edge

import (
	"context"
	"fmt"

	"github.com/flocko-motion/rankedb/go/cmd/ranke-cli/internal/cli"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge <id>",
		Short: "Show an edge by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cli.Client()
			if err != nil {
				return err
			}
			ctx := context.Background()
			jsonOut, _ := cmd.Flags().GetBool("json")

			resp, err := c.GetApiEdgesIdWithResponse(ctx, args[0])
			if err != nil {
				return err
			}
			if resp.JSON200 == nil {
				return fmt.Errorf("edge not found (status %d)", resp.StatusCode())
			}

			if jsonOut {
				cli.PrintJSON(resp.JSON200)
			} else {
				cli.PrintEdge(*resp.JSON200)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}
