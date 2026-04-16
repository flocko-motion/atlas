package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rankedb "github.com/flocko-motion/rankedb-sdk"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ranke-ingest-bulk <format> <archive-file>",
		Short: "Upload an archive file to RankeDB as a source/bulk node",
		Long: `Uploads an archive file (.tar.gz, .tgz, .zip) to RankeDB as an L0 source/bulk root node.

The format argument tells unpack workers what kind of archive this is
(e.g. google-takeout, whatsapp-export, signal-backup).

Examples:
  ranke-ingest-bulk google-takeout takeout.tgz
  ranke-ingest-bulk whatsapp-export chat.zip`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			server, _ := cmd.Flags().GetString("server")
			origin, _ := cmd.Flags().GetString("origin")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			return run(args[1], server, args[0], origin, dryRun)
		},
	}

	rootCmd.Flags().String("server", "http://localhost:7000", "RankeDB server URL")
	rootCmd.Flags().String("origin", "", "Origin label (default: derived from filename)")
	rootCmd.Flags().Bool("dry-run", false, "Show what would be uploaded without sending")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(path string, server string, format string, origin string, dryRun bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory — this tool expects an archive file (.tar.gz, .tgz, .zip)", path)
	}

	const maxSize = 100 * 1024 * 1024 // 100 MB
	if info.Size() > maxSize {
		return fmt.Errorf("%s is %s — exceeds 100 MB limit (large archives should be unpacked before upload)", path, humanSize(info.Size()))
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	encodingClass, encodingFormat := detectEncoding(path)
	originalName := filepath.Base(path)

	hash := sha256.Sum256(content)
	sha := hex.EncodeToString(hash[:])

	if origin == "" {
		origin = "bulk-ingest/" + originalName
	}

	fmt.Printf("File:     %s\n", originalName)
	fmt.Printf("Size:     %s\n", humanSize(int64(len(content))))
	fmt.Printf("SHA-256:  %s\n", sha)
	fmt.Printf("Encoding: %s/%s\n", encodingClass, encodingFormat)
	fmt.Printf("Format:   %s\n", format)
	fmt.Printf("Origin:   %s\n", origin)

	if dryRun {
		fmt.Println("\n(dry run — nothing uploaded)")
		return nil
	}

	fmt.Printf("Server:   %s\n\n", server)

	client, err := rankedb.NewClient(server)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	// L0 roots are idempotent by content hash — the server handles dedup
	ctx := context.Background()
	contentStr := string(content)
	node, err := client.CreateNode(ctx, rankedb.CreateNodeRequest{
		Level:          0,
		ContentClass:   "source",
		ContentType:    "bulk",
		EncodingClass:  encodingClass,
		EncodingFormat: encodingFormat,
		Content:        &contentStr,
		Origin:         rankedb.Ptr(origin),
		OriginalName:   rankedb.Ptr(originalName),
	})
	if err != nil {
		return fmt.Errorf("create node: %w", err)
	}

	fmt.Printf("Created node: %s\n", node.Id)
	return nil
}

func detectEncoding(path string) (string, string) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".tgz":
		return "application", "tar+gzip"
	case ".gz":
		return "application", "tar+gzip"
	case ".zip":
		return "application", "zip"
	case ".tar":
		return "application", "tar"
	default:
		return "application", "octet-stream"
	}
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
