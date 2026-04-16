package relations

import (
	"context"
	"fmt"

	"github.com/flocko-motion/rankedb/cmd/ranke-cli/internal/cli"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relations",
		Short: "List relations (L2)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cli.Client()
			if err != nil {
				return err
			}
			ctx := context.Background()
			jsonOut, _ := cmd.Flags().GetBool("json")

			resp, err := c.GetApiRelationsWithResponse(ctx)
			if err != nil {
				return err
			}
			if resp.JSON200 == nil {
				return fmt.Errorf("unexpected status: %d", resp.StatusCode())
			}

			if jsonOut {
				cli.PrintJSON(resp.JSON200.Relations)
			} else {
				for _, r := range resp.JSON200.Relations {
					cli.PrintRelation(r)
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}
