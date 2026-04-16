package entity

import (
	"context"
	"fmt"

	"rankedb/cmd/ranke-cli/internal/cli"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entity <id>",
		Short: "Show an entity with its relations",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cli.Client()
			if err != nil {
				return err
			}
			ctx := context.Background()
			jsonOut, _ := cmd.Flags().GetBool("json")
			timeline, _ := cmd.Flags().GetBool("timeline")

			if timeline {
				resp, err := c.GetApiEntitiesIdTimelineWithResponse(ctx, args[0])
				if err != nil {
					return err
				}
				if resp.JSON200 == nil {
					return fmt.Errorf("entity not found (status %d)", resp.StatusCode())
				}
				if jsonOut {
					cli.PrintJSON(resp.JSON200.Relations)
				} else {
					for _, r := range resp.JSON200.Relations {
						cli.PrintRelation(r)
					}
				}
				return nil
			}

			resp, err := c.GetApiEntitiesIdWithResponse(ctx, args[0])
			if err != nil {
				return err
			}
			if resp.JSON200 == nil {
				return fmt.Errorf("entity not found (status %d)", resp.StatusCode())
			}

			if jsonOut {
				cli.PrintJSON(resp.JSON200)
			} else {
				cli.PrintNode(resp.JSON200.Entity)
				if len(resp.JSON200.Relations) > 0 {
					fmt.Printf("\nRelations:\n")
					for _, r := range resp.JSON200.Relations {
						cli.PrintRelation(r)
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	cmd.Flags().BoolP("timeline", "t", false, "Show relations as timeline")
	return cmd
}
