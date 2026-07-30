// package: main / cmd
// type:    entrypoint
// job:     the ranke-db binary — a cobra CLI handing a config to the config package
// limits:  CLI wiring only; decrypt/parse/resolve/assemble live in config (-> config)
//
// Command ranke-db is the single binary. It opens the launch artifact (a file or
// stdin), builds an age PassphraseSource from --age-key (prompt|stdin|env:VAR|
// file:path — never a command-line literal), and hands both to config: "run"
// assembles the stack and serves until SIGINT/SIGTERM; "verify" checks the config
// to a chosen depth and exits. The config package owns everything past the
// handoff; the CLI only generalises its inputs into config's two entry points.
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/flocko-motion/rankedb/config"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "ranke-db:", err)
		os.Exit(1)
	}
}

// rootCmd builds the ranke-db command tree. Silence* keeps cobra from dumping
// usage and a second error line on a runtime failure — main prints the error.
func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "ranke-db",
		Short:         "Serve a Ranke-Graph from a launch artifact",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(runCmd(), verifyCmd())
	return root
}

// runCmd assembles the stack from a config and serves it.
func runCmd() *cobra.Command {
	var ageKey string
	c := &cobra.Command{
		Use:   "run [flags] <configfile>|-",
		Short: "Assemble the adapter stack from a config and serve it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, cleanup, err := openConfig(args[0], ageKey)
			if err != nil {
				return err
			}
			defer cleanup()
			src, err := passphraseFrom(ageKey, os.Stdin)
			if err != nil {
				return err
			}
			app, err := config.Run(cmd.Context(), cfg, src)
			if err != nil {
				return err
			}
			if err := requireServing(app); err != nil {
				return err
			}
			return serve(app)
		},
	}
	c.Flags().StringVar(&ageKey, "age-key", "", "age key source: prompt|stdin|env:VAR|file:path")
	return c
}

// verifyCmd checks a config to the chosen depth and reports, without serving.
func verifyCmd() *cobra.Command {
	var ageKey, levelName string
	c := &cobra.Command{
		Use:   "verify [flags] <configfile>|-",
		Short: "Check a config to a chosen depth (syntax|resolve)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			level, err := parseLevel(levelName)
			if err != nil {
				return err
			}
			cfg, cleanup, err := openConfig(args[0], ageKey)
			if err != nil {
				return err
			}
			defer cleanup()
			src, err := passphraseFrom(ageKey, os.Stdin)
			if err != nil {
				return err
			}
			if err := config.Verify(cmd.Context(), cfg, src, level); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "config ok")
			return nil
		},
	}
	c.Flags().StringVar(&ageKey, "age-key", "", "age key source: prompt|stdin|env:VAR|file:path")
	c.Flags().StringVar(&levelName, "level", "syntax", "verify depth: syntax|resolve|connect")
	return c
}

// openConfig opens the launch artifact as a reader: a file (closed by cleanup)
// or stdin. Reading both the config and the age key from stdin is refused.
func openConfig(path, ageKey string) (io.Reader, func(), error) {
	if path == "-" {
		if ageKey == "stdin" {
			return nil, nil, errors.New("cannot read both the config and the age key from stdin; use --age-key prompt|env:VAR|file:path")
		}
		return os.Stdin, func() {}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open config: %w", err)
	}
	return f, func() { _ = f.Close() }, nil
}

// parseLevel maps the --level flag to a config.Level.
func parseLevel(s string) (config.Level, error) {
	switch s {
	case "syntax", "":
		return config.LevelSyntax, nil
	case "resolve":
		return config.LevelResolve, nil
	case "connect":
		return config.LevelConnect, nil
	default:
		return 0, fmt.Errorf("unknown --level %q (want syntax|resolve|connect)", s)
	}
}

// passphraseFrom builds a config.PassphraseSource from a spec: prompt, stdin,
// env:VAR, file:path, or empty for none. A literal passphrase is deliberately
// unsupported — never pass the key as a command-line argument.
func passphraseFrom(spec string, in io.Reader) (config.PassphraseSource, error) {
	switch {
	case spec == "":
		return nil, nil
	case spec == "prompt":
		return promptPassphrase, nil
	case spec == "stdin":
		return func() (string, error) {
			b, err := io.ReadAll(in)
			if err != nil {
				return "", fmt.Errorf("read age passphrase from stdin: %w", err)
			}
			return strings.TrimRight(string(b), "\r\n"), nil
		}, nil
	case strings.HasPrefix(spec, "env:"):
		name := strings.TrimPrefix(spec, "env:")
		return func() (string, error) {
			v, ok := os.LookupEnv(name)
			if !ok {
				return "", fmt.Errorf("age key env(%s) is not set", name)
			}
			return v, nil
		}, nil
	case strings.HasPrefix(spec, "file:"):
		path := strings.TrimPrefix(spec, "file:")
		return func() (string, error) {
			b, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("read age key file: %w", err)
			}
			return strings.TrimRight(string(b), "\r\n"), nil
		}, nil
	default:
		return nil, fmt.Errorf("unknown age key source %q; use prompt|stdin|env:VAR|file:path", spec)
	}
}

// promptPassphrase reads from the controlling terminal without echo, opening /dev/tty
// directly so it works when stdin is the config pipe.
func promptPassphrase() (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("open terminal for passphrase prompt: %w", err)
	}
	defer func() { _ = tty.Close() }()
	fmt.Fprint(tty, "age passphrase: ")
	b, err := term.ReadPassword(int(tty.Fd()))
	fmt.Fprintln(tty)
	if err != nil {
		return "", fmt.Errorf("read passphrase: %w", err)
	}
	return string(b), nil
}

// requireServing enforces what serving cannot do without: a signer, storage, a
// sequencer to reach the archive through, and an endpoint to reach them. Run assembles
// only what is configured; the policy lives here.
func requireServing(app *config.App) error {
	var missing []string
	if app.Signer == nil {
		missing = append(missing, "signer")
	}
	if app.Storage == nil {
		missing = append(missing, "storage")
	}
	if app.Sequencer == nil {
		// GetArchive is the sequencer's, and an archive is what every read opens, so a
		// stack without one answers nothing.
		missing = append(missing, "sequencer")
	}
	if len(app.Endpoints) == 0 {
		missing = append(missing, "endpoints")
	}
	if len(missing) > 0 {
		return fmt.Errorf("config is missing required section(s) for serving: %v", missing)
	}
	return nil
}

// serve runs every endpoint the config mounted, concurrently, until the process is
// signalled. Each listens where its own section says: no flag can disagree.
func serve(app *config.App) error {
	logIdentity(app)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	sctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errc := make(chan error, len(app.Endpoints))
	var wg sync.WaitGroup
	for _, ep := range app.Endpoints {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errc <- ep.Serve(sctx)
		}()
	}
	slog.Info("ranke-db serving", "endpoints", len(app.Endpoints))

	// The first endpoint to fail takes the process down: a stack serving less than
	// configured is not the stack the operator asked for.
	select {
	case err := <-errc:
		cancel()
		wg.Wait()
		return err
	case <-ctx.Done():
		slog.Info("ranke-db shutting down")
		cancel()
		wg.Wait()
		return nil
	}
}

// logIdentity reports which key the server attests merges under.
func logIdentity(app *config.App) {
	pub, err := app.Signer.Public(context.Background())
	if err != nil {
		slog.Warn("ranke-db: could not read signer identity", "err", err)
		return
	}
	id := fmt.Sprintf("%T", pub)
	if ed, ok := pub.(ed25519.PublicKey); ok {
		id = "ed25519:" + base64.RawStdEncoding.EncodeToString(ed)
	}
	slog.Info("ranke-db assembled", "signer", id)
}
