package entities

import (
	"context"
	"fmt"

	"github.com/flocko-motion/rankedb/cmd/ranke-cli/internal/cli"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entities",
		Short: "List entities (L2)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cli.Client()
			if err != nil {
				return err
			}
			ctx := context.Background()
			jsonOut, _ := cmd.Flags().GetBool("json")

			resp, err := c.GetApiEntitiesWithResponse(ctx)
			if err != nil {
				return err
			}
			if resp.JSON200 == nil {
				return fmt.Errorf("unexpected status: %d", resp.StatusCode())
			}

			if jsonOut {
				cli.PrintJSON(resp.JSON200.Entities)
			} else {
				cli.PrintNodeTable(resp.JSON200.Entities)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}
