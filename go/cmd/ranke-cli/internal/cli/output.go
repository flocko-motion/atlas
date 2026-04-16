package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/flocko-motion/rankedb/go/apiclient"
)

func PrintJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func PrintNode(n apiclient.NodeResponse) {
	fmt.Printf("ID:       %s\n", n.Id)
	fmt.Printf("Level:    %d\n", n.Level)
	fmt.Printf("Class:    %s/%s\n", n.ContentClass, n.ContentType)
	fmt.Printf("Encoding: %s/%s\n", n.EncodingClass, n.EncodingFormat)
	fmt.Printf("Size:     %d bytes\n", n.ContentLen)
	fmt.Printf("SHA-256:  %s\n", n.ContentSha256)
	fmt.Printf("Created:  %s\n", n.CreatedAt)
	if n.Origin != "" {
		fmt.Printf("Origin:   %s\n", n.Origin)
	}
	if n.OriginalName != "" {
		fmt.Printf("Name:     %s\n", n.OriginalName)
	}
	if n.ArtifactCreatedAt != "" {
		fmt.Printf("Artifact: %s\n", n.ArtifactCreatedAt)
	}
	if n.Confidence != 0 {
		fmt.Printf("Conf:     %.2f\n", n.Confidence)
	}
	if n.ValidFrom != "" {
		fmt.Printf("Valid:    %s — %s\n", n.ValidFrom, n.ValidUntil)
	}
	if n.Content != "" {
		content := n.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		fmt.Printf("\n%s\n", content)
	}
}

func PrintNodeTable(nodes []apiclient.NodeResponse) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLEVEL\tCLASS/TYPE\tENCODING\tSIZE\tCREATED")
	for _, n := range nodes {
		id := n.Id
		if len(id) > 20 {
			id = id[:20] + "..."
		}
		fmt.Fprintf(w, "%s\tL%d\t%s/%s\t%s/%s\t%d\t%s\n",
			id, n.Level,
			n.ContentClass, n.ContentType,
			n.EncodingClass, n.EncodingFormat,
			n.ContentLen,
			shortTime(n.CreatedAt),
		)
	}
	w.Flush()
}

func PrintEdge(e apiclient.EdgeResponse) {
	fmt.Printf("%s  %s -> %s  [%s]", e.Id, e.SourceNodeId, e.TargetNodeId, e.Type)
	if e.Confidence != 0 {
		fmt.Printf("  conf=%.2f", e.Confidence)
	}
	fmt.Println()
}

func PrintEdgeTable(edges []apiclient.EdgeResponse) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSOURCE\tTARGET\tTYPE\tCONF")
	for _, e := range edges {
		conf := ""
		if e.Confidence != 0 {
			conf = fmt.Sprintf("%.2f", e.Confidence)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			Shorten(e.Id), Shorten(e.SourceNodeId), Shorten(e.TargetNodeId), e.Type, conf)
	}
	w.Flush()
}

func PrintRelation(r apiclient.RelationResponse) {
	tails := make([]string, len(r.TailEdges))
	for i, e := range r.TailEdges {
		tails[i] = Shorten(e.SourceNodeId)
	}
	heads := make([]string, len(r.HeadEdges))
	for i, e := range r.HeadEdges {
		heads[i] = Shorten(e.TargetNodeId)
	}
	label := r.Node.Content
	if label == "" {
		label = r.Node.ContentType
	}
	fmt.Printf("[%s] --%s--> [%s]  (%s)\n",
		strings.Join(tails, ", "), label, strings.Join(heads, ", "), Shorten(r.Node.Id))
}

func Shorten(id string) string {
	if len(id) > 20 {
		return id[:20] + "..."
	}
	return id
}

func shortTime(t string) string {
	if len(t) > 19 {
		return t[:19]
	}
	return t
}
