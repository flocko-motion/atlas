package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"rankedb/apiclient"
)

func client() (*apiclient.ClientWithResponses, error) {
	return apiclient.NewClientWithResponses(cfg.Server)
}

// --- status ---

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check server connectivity and show basic stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Quick connectivity check
			httpClient := &http.Client{Timeout: 5 * time.Second}
			resp, err := httpClient.Get(cfg.Server + "/api/nodes?limit=0")
			if err != nil {
				return fmt.Errorf("cannot reach %s: %w", cfg.Server, err)
			}
			resp.Body.Close()
			fmt.Printf("Server:  %s\n", cfg.Server)
			fmt.Printf("Status:  %s\n", resp.Status)

			// Count nodes by level
			c, err := client()
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

// --- nodes ---

func nodesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "List nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
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
				printJSON(resp.JSON200.Nodes)
			} else {
				printNodeTable(resp.JSON200.Nodes)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

// --- node <id> ---

func nodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node <id>",
		Short: "Show a node by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
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
				printJSON(resp.JSON200)
			} else {
				printNode(*resp.JSON200)
			}

			if showEdges {
				edgesResp, err := c.GetApiNodesIdEdgesWithResponse(ctx, args[0])
				if err != nil {
					return err
				}
				if edgesResp.JSON200 != nil && len(edgesResp.JSON200.Edges) > 0 {
					fmt.Printf("\nEdges:\n")
					printEdgeTable(edgesResp.JSON200.Edges)
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	cmd.Flags().BoolP("edges", "e", false, "Also show connected edges")
	return cmd
}

// --- entities ---

func entitiesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entities",
		Short: "List entities (L2)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
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
				printJSON(resp.JSON200.Entities)
			} else {
				printNodeTable(resp.JSON200.Entities)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

// --- entity <id> ---

func entityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entity <id>",
		Short: "Show an entity with its relations",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
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
					printJSON(resp.JSON200.Relations)
				} else {
					for _, r := range resp.JSON200.Relations {
						printRelation(r)
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
				printJSON(resp.JSON200)
			} else {
				printNode(resp.JSON200.Entity)
				if len(resp.JSON200.Relations) > 0 {
					fmt.Printf("\nRelations:\n")
					for _, r := range resp.JSON200.Relations {
						printRelation(r)
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

// --- relations ---

func relationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relations",
		Short: "List relations (L2)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
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
				printJSON(resp.JSON200.Relations)
			} else {
				for _, r := range resp.JSON200.Relations {
					printRelation(r)
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

// --- edge <id> ---

func edgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge <id>",
		Short: "Show an edge by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
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
				printJSON(resp.JSON200)
			} else {
				printEdge(*resp.JSON200)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

// --- provenance <node-id> ---

func provenanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provenance <node-id>",
		Short: "Show the provenance chain of a node back to L0 roots",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
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
				printJSON(resp.JSON200)
				return nil
			}

			fmt.Printf("Provenance chain for %s\n\n", args[0])
			fmt.Printf("Nodes (%d):\n", len(resp.JSON200.Nodes))
			printNodeTable(resp.JSON200.Nodes)
			fmt.Printf("\nEdges (%d):\n", len(resp.JSON200.Edges))
			printEdgeTable(resp.JSON200.Edges)
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}
