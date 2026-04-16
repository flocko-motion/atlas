package nodes

import (
	"context"
	"fmt"

	"rankedb/cmd/ranke-cli/internal/cli"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "List nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cli.Client()
			if err != nil {
				return err
			}
			ctx := context.Background()
			jsonOut, _ := cmd.Flags().GetBool("json")

			resp, err := c.GetApiNodesWithResponse(ctx)
			if err != nil {
				return err
			}
			if resp.JSON200 == nil {
				return fmt.Errorf("unexpected status: %d", resp.StatusCode())
			}

			if jsonOut {
				cli.PrintJSON(resp.JSON200.Nodes)
			} else {
				cli.PrintNodeTable(resp.JSON200.Nodes)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}
