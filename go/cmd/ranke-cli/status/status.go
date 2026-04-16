package status

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/flocko-motion/rankedb/cmd/ranke-cli/internal/cli"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check server connectivity and show basic stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			httpClient := &http.Client{Timeout: 5 * time.Second}
			resp, err := httpClient.Get(cli.Cfg.Server + "/api/nodes?limit=0")
			if err != nil {
				return fmt.Errorf("cannot reach %s: %w", cli.Cfg.Server, err)
			}
			resp.Body.Close()
			fmt.Printf("Server:  %s\n", cli.Cfg.Server)
			fmt.Printf("Status:  %s\n", resp.Status)

			c, err := cli.Client()
			if err != nil {
				return err
			}
			ctx := context.Background()

			nodesResp, err := c.GetApiNodesWithResponse(ctx)
			if err != nil {
				return err
			}
			if nodesResp.JSON200 != nil {
				nodes := nodesResp.JSON200.Nodes
				l0, l1, l2 := 0, 0, 0
				for _, n := range nodes {
					switch n.Level {
					case 0:
						l0++
					case 1:
						l1++
					case 2:
						l2++
					}
				}
				fmt.Printf("\nNodes:   %d (L0: %d, L1: %d, L2: %d)\n", len(nodes), l0, l1, l2)
			}
			return nil
		},
	}
}
