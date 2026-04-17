// Package ingest implements the `ranke-cli ingest <path>` command.
// If <path> is a file, uploads it as a single L0 source node.
// If <path> is a directory, walks it and uploads each matching file.
// Content-hash precheck (GET /api/nodes/{sha}) skips files already present.
package ingest

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"rankedb/cmd/ranke-cli/internal/cli"
	"rankedb/cmd/ranke-cli/internal/encoding"

	"github.com/spf13/cobra"
)

const maxFileSize = 100 * 1024 * 1024 // 100 MB per file

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest <path>...",
		Short: "Upload files or every matching file in directories as L0 source nodes",
		Long: `Uploads one or many files as L0 source nodes.

Each argument may be a file or a directory. Shell globs are fine — they expand
to multiple file arguments. File arguments are uploaded directly. Directory
arguments are walked with --match applied to filenames.

Source type and encoding class/format are inferred per-file from the extension:
  .eml → source/conversation, text/eml
  .jpg → source/media, image/jpeg
  .tgz → source/bulk, application/tar+gzip
  .vcf → source/contact, text/vcf
  ...and so on. Flags --type, --format, --encoding-class override the inference.

Files with unrecognized extensions or ambiguous types (e.g. .txt, .md) are
skipped unless the missing pieces are provided via flags. Every upload is
preceded by a SHA-256 precheck, so re-running over the same tree only uploads
new files.

Examples:
  ranke-cli ingest takeout.tgz
  ranke-cli ingest ./0000*.eml --origin=gmail
  ranke-cli ingest ~/mail --match='*.eml'
  ranke-cli ingest ./photos --match='*.jpg'
  ranke-cli ingest ./notes --match='*.md' --type=data`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := options{
				paths:         args,
				typ:           mustString(cmd, "type"),
				encodingClass: mustString(cmd, "encoding-class"),
				format:        mustString(cmd, "format"),
				match:         mustString(cmd, "match"),
				origin:        mustString(cmd, "origin"),
				title:         mustString(cmd, "title"),
				concurrency:   mustInt(cmd, "concurrency"),
				dryRun:        mustBool(cmd, "dry-run"),
			}
			if opts.concurrency < 1 {
				opts.concurrency = 1
			}
			return run(opts)
		},
	}
	cmd.Flags().String("type", "", "Source content type override (default: inferred from extension)")
	cmd.Flags().String("encoding-class", "", "Encoding class override (default: inferred from extension)")
	cmd.Flags().String("format", "", "Encoding format override (default: inferred from extension)")
	cmd.Flags().String("match", "*", "Glob pattern for filenames (directory mode)")
	cmd.Flags().String("origin", "", "Origin label (default: derived from path)")
	cmd.Flags().String("title", "", "Node title (single-file mode only)")
	cmd.Flags().Int("concurrency", 8, "Parallel upload workers (directory mode)")
	cmd.Flags().Bool("dry-run", false, "Show what would be uploaded without sending")
	return cmd
}

type options struct {
	paths         []string
	typ           string
	encodingClass string // override
	format        string // override
	match         string
	origin        string
	title         string
	concurrency   int
	dryRun        bool
}

type stats struct {
	found     atomic.Int64
	unknown   atomic.Int64 // extension not recognized, no override
	tooLarge  atomic.Int64
	existing  atomic.Int64 // SHA precheck hit
	uploaded  atomic.Int64
	failed    atomic.Int64
	bytesUp   atomic.Int64
}

type source struct {
	path string
	dir  bool
}

func run(opts options) error {
	srcs := make([]source, 0, len(opts.paths))
	for _, p := range opts.paths {
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("stat %s: %w", p, err)
		}
		srcs = append(srcs, source{path: p, dir: info.IsDir()})
	}

	// Single-file invocation keeps the pretty per-node output.
	if len(srcs) == 1 && !srcs[0].dir {
		info, _ := os.Stat(srcs[0].path)
		return runSingle(opts, srcs[0].path, info)
	}
	return runBatch(srcs, opts)
}

// ─── Single-file mode ─────────────────────────────────────────────────────────

func runSingle(opts options, path string, info os.FileInfo) error {
	if info.Size() > maxFileSize {
		return fmt.Errorf("%s is %s — exceeds %s limit", path, humanSize(info.Size()), humanSize(maxFileSize))
	}

	encClass, encFmt, typ, missing := resolve(path, opts)
	if missing != "" {
		return fmt.Errorf("cannot ingest %s: %s (pass the corresponding flag to override)", path, missing)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	hash := sha256.Sum256(content)
	sha := hex.EncodeToString(hash[:])

	originalName := filepath.Base(path)
	origin := opts.origin
	if origin == "" {
		origin = "ingest/" + originalName
	}

	fmt.Printf("File:     %s\n", originalName)
	fmt.Printf("Size:     %s\n", humanSize(int64(len(content))))
	fmt.Printf("SHA-256:  %s\n", sha)
	fmt.Printf("Type:     source/%s\n", typ)
	fmt.Printf("Encoding: %s/%s\n", encClass, encFmt)
	fmt.Printf("Origin:   %s\n", origin)

	if opts.dryRun {
		fmt.Println("\n(dry run — nothing uploaded)")
		return nil
	}

	exists, err := shaExists(sha)
	if err != nil {
		return fmt.Errorf("sha precheck: %w", err)
	}
	if exists {
		fmt.Println("\nAlready present — skipping upload.")
		return nil
	}

	fmt.Printf("Server:   %s\n\n", cli.Cfg.Server)
	nodeID, err := uploadFile(originalName, content, typ, encClass, encFmt, origin, opts.title)
	if err != nil {
		return err
	}
	fmt.Printf("Created node: %s\n", nodeID)
	return nil
}

// ─── Batch mode (multi-arg and/or directory walk) ────────────────────────────

func runBatch(srcs []source, opts options) error {
	origin := opts.origin
	if origin == "" {
		// Derive from the first source — a dir basename or the file's parent dir.
		base := filepath.Base(filepath.Clean(srcs[0].path))
		if !srcs[0].dir {
			base = filepath.Base(filepath.Dir(srcs[0].path))
			if base == "." || base == "/" {
				base = "batch"
			}
		}
		origin = "ingest/" + base
	}

	roots := make([]string, len(srcs))
	for i, s := range srcs {
		roots[i] = s.path
	}
	fmt.Printf("Sources:      %s\n", strings.Join(roots, " "))
	fmt.Printf("Match:        %s (applies to directory walks only)\n", opts.match)
	if opts.typ != "" {
		fmt.Printf("Type:         source/%s (override)\n", opts.typ)
	} else {
		fmt.Printf("Type:         auto (per-file)\n")
	}
	if opts.encodingClass != "" || opts.format != "" {
		fmt.Printf("Encoding:     %s/%s (override)\n", firstNonEmpty(opts.encodingClass, "auto"), firstNonEmpty(opts.format, "auto"))
	} else {
		fmt.Printf("Encoding:     auto (per-file)\n")
	}
	fmt.Printf("Origin:       %s\n", origin)
	fmt.Printf("Concurrency:  %d\n", opts.concurrency)
	fmt.Printf("Server:       %s\n", cli.Cfg.Server)
	if opts.dryRun {
		fmt.Println("Mode:         dry-run")
	}
	fmt.Println()

	paths := make(chan string, opts.concurrency*4)
	var s stats
	var wg sync.WaitGroup
	for i := 0; i < opts.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range paths {
				processFile(p, opts, origin, &s)
			}
		}()
	}

	done := make(chan struct{})
	go progressTicker(&s, done)

	var walkErrs []error
	for _, src := range srcs {
		if !src.dir {
			// Explicit file arg — skip --match filter, the user chose this file.
			s.found.Add(1)
			paths <- src.path
			continue
		}
		err := filepath.Walk(src.path, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				return nil
			}
			matched, err := filepath.Match(opts.match, fi.Name())
			if err != nil {
				return err
			}
			if !matched {
				return nil
			}
			s.found.Add(1)
			paths <- path
			return nil
		})
		if err != nil {
			walkErrs = append(walkErrs, fmt.Errorf("walk %s: %w", src.path, err))
		}
	}
	close(paths)
	wg.Wait()
	close(done)

	for _, e := range walkErrs {
		fmt.Fprintf(os.Stderr, "%v\n", e)
	}

	fmt.Printf("\nFound:    %d\n", s.found.Load())
	fmt.Printf("Existing: %d (SHA precheck)\n", s.existing.Load())
	fmt.Printf("Uploaded: %d (%s)\n", s.uploaded.Load(), humanSize(s.bytesUp.Load()))
	fmt.Printf("Unknown:  %d (unrecognized extension, skipped)\n", s.unknown.Load())
	fmt.Printf("TooLarge: %d (skipped, > %s)\n", s.tooLarge.Load(), humanSize(maxFileSize))
	fmt.Printf("Failed:   %d\n", s.failed.Load())

	if s.failed.Load() > 0 {
		return fmt.Errorf("%d upload(s) failed", s.failed.Load())
	}
	return nil
}

func processFile(path string, opts options, origin string, s *stats) {
	fi, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stat %s: %v\n", path, err)
		s.failed.Add(1)
		return
	}
	if fi.Size() > maxFileSize {
		fmt.Fprintf(os.Stderr, "skipping %s: exceeds %s limit (%s)\n", path, humanSize(maxFileSize), humanSize(fi.Size()))
		s.tooLarge.Add(1)
		return
	}

	encClass, encFmt, typ, missing := resolve(path, opts)
	if missing != "" {
		fmt.Fprintf(os.Stderr, "skipping %s: %s\n", path, missing)
		s.unknown.Add(1)
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
		s.failed.Add(1)
		return
	}

	hash := sha256.Sum256(content)
	sha := hex.EncodeToString(hash[:])

	exists, err := shaExists(sha)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sha precheck %s: %v\n", path, err)
		s.failed.Add(1)
		return
	}
	if exists {
		s.existing.Add(1)
		return
	}

	if opts.dryRun {
		fmt.Printf("[dry-run] %s (%s, source/%s, %s/%s, sha=%s…)\n", path, humanSize(fi.Size()), typ, encClass, encFmt, sha[:12])
		s.uploaded.Add(1)
		s.bytesUp.Add(fi.Size())
		return
	}

	if _, err := uploadFile(filepath.Base(path), content, typ, encClass, encFmt, origin, ""); err != nil {
		fmt.Fprintf(os.Stderr, "upload %s: %v\n", path, err)
		s.failed.Add(1)
		return
	}
	s.uploaded.Add(1)
	s.bytesUp.Add(fi.Size())
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// resolve picks encoding class/format and source type for a file, merging
// autodetect results with flag overrides. Returns a non-empty `missing` string
// describing what couldn't be resolved (used as the skip/error reason).
func resolve(path string, opts options) (class, format, typ, missing string) {
	detClass, detFormat, detTyp, _ := encoding.Detect(path)
	class = firstNonEmpty(opts.encodingClass, detClass)
	format = firstNonEmpty(opts.format, detFormat)
	typ = firstNonEmpty(opts.typ, detTyp)
	var miss []string
	if class == "" {
		miss = append(miss, "unknown encoding class")
	}
	if format == "" {
		miss = append(miss, "unknown encoding format")
	}
	if typ == "" {
		miss = append(miss, "ambiguous source type")
	}
	if len(miss) > 0 {
		missing = strings.Join(miss, ", ")
	}
	return class, format, typ, missing
}

func shaExists(sha string) (bool, error) {
	resp, err := http.Get(cli.Cfg.Server + "/api/nodes/" + sha)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		b, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
}

func uploadFile(originalName string, content []byte, typ, encClass, encFmt, origin, title string) (string, error) {
	var payload string
	if encClass == "text" {
		payload = string(content)
	} else {
		payload = base64.StdEncoding.EncodeToString(content)
	}
	body := map[string]any{
		"level":           0,
		"content_class":   "source",
		"content_type":    typ,
		"encoding_class":  encClass,
		"encoding_format": encFmt,
		"content":         payload,
		"origin":          origin,
		"original_name":   originalName,
	}
	if title != "" {
		body["title"] = title
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	resp, err := http.Post(cli.Cfg.Server+"/api/nodes", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	var node struct {
		Id string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&node)
	return node.Id, nil
}

func progressTicker(s *stats, done <-chan struct{}) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-done:
			return
		case <-t.C:
			fmt.Fprintf(os.Stderr, "  …found=%d existing=%d uploaded=%d unknown=%d failed=%d\n",
				s.found.Load(), s.existing.Load(), s.uploaded.Load(), s.unknown.Load(), s.failed.Load())
		}
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

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
func mustInt(cmd *cobra.Command, name string) int {
	v, _ := cmd.Flags().GetInt(name)
	return v
}
func mustBool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}
