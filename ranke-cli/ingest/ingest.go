package ingest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/flocko-motion/ranke-cli/internal/cli"
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest <format> <archive-file>",
		Short: "Upload a bulk archive to RankeDB",
		Long: `Uploads an archive file (.tar.gz, .tgz, .zip) as an L0 source/bulk root node.

The format argument tells unpack workers what kind of archive this is
(e.g. google-takeout, whatsapp-export, signal-backup).

Examples:
  ranke-cli ingest google-takeout takeout.tgz
  ranke-cli ingest whatsapp-export chat.zip`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			origin, _ := cmd.Flags().GetString("origin")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			return runIngest(args[0], args[1], origin, dryRun)
		},
	}
	cmd.Flags().String("origin", "", "Origin label (default: derived from filename)")
	cmd.Flags().Bool("dry-run", false, "Show what would be uploaded without sending")
	return cmd
}

func runIngest(format string, path string, origin string, dryRun bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory — expected an archive file (.tar.gz, .tgz, .zip)", path)
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

	fmt.Printf("Server:   %s\n\n", cli.Cfg.Server)

	contentStr := string(content)
	reqBody, _ := json.Marshal(map[string]any{
		"level":           0,
		"content_class":   "source",
		"content_type":    "bulk",
		"encoding_class":  encodingClass,
		"encoding_format": encodingFormat,
		"content":         contentStr,
		"origin":          origin,
		"original_name":   originalName,
	})

	resp, err := http.Post(cli.Cfg.Server+"/api/nodes", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("POST /api/nodes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(b))
	}

	var node struct {
		Id string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&node)
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
