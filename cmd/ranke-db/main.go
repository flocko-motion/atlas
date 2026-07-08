// package: main / cmd
// type:    entrypoint
// job:     the ranke-db binary — "run"/"verify" a config: open it, build the age key source, hand off to config
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
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/flocko-motion/rankedb/config"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ranke-db:", err)
		os.Exit(1)
	}
}

// run dispatches the subcommand.
func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "run":
		return cmdRun(args[1:])
	case "verify":
		return cmdVerify(args[1:])
	case "-h", "--help", "help":
		return usage()
	default:
		return fmt.Errorf("unknown command %q (try: ranke-db run <configfile>)", args[0])
	}
}

func usage() error {
	fmt.Fprint(os.Stderr, `ranke-db — serve a Ranke-Graph from a launch artifact

usage:
  ranke-db run    [--addr ADDR] [--age-key SRC] <configfile>|-
  ranke-db verify [--level L]   [--age-key SRC] <configfile>|-

  <configfile>|-   path to the JSON launch artifact, or - to read from stdin
  --addr ADDR      address to serve on (default :8080)
  --level L        verify depth: syntax (default) | resolve
  --age-key SRC    passphrase source for an age-encrypted config:
                   prompt | stdin | env:VAR | file:path
`)
	return nil
}

// cmdRun implements "ranke-db run": open the config, build the key source, hand
// to config.Run, then serve the assembled stack.
func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	addr := fs.String("addr", ":8080", "address to serve on")
	ageKey := fs.String("age-key", "", "age key source: prompt|stdin|env:VAR|file:path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: ranke-db run [--addr ADDR] [--age-key SRC] <configfile>|-")
	}

	cfg, cleanup, err := openConfig(fs.Arg(0), *ageKey)
	if err != nil {
		return err
	}
	defer cleanup()
	src, err := passphraseFrom(*ageKey, os.Stdin)
	if err != nil {
		return err
	}

	app, err := config.Run(context.Background(), cfg, src)
	if err != nil {
		return err
	}
	if err := requireServing(app); err != nil {
		return err
	}
	return serve(*addr, app)
}

// cmdVerify implements "ranke-db verify": check the config to the chosen depth
// and report, without serving.
func cmdVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	ageKey := fs.String("age-key", "", "age key source: prompt|stdin|env:VAR|file:path")
	levelName := fs.String("level", "syntax", "verify depth: syntax|resolve")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: ranke-db verify [--level L] [--age-key SRC] <configfile>|-")
	}
	level, err := parseLevel(*levelName)
	if err != nil {
		return err
	}

	cfg, cleanup, err := openConfig(fs.Arg(0), *ageKey)
	if err != nil {
		return err
	}
	defer cleanup()
	src, err := passphraseFrom(*ageKey, os.Stdin)
	if err != nil {
		return err
	}

	if err := config.Verify(context.Background(), cfg, src, level); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "config ok")
	return nil
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
	default:
		return 0, fmt.Errorf("unknown --level %q (want syntax|resolve)", s)
	}
}

// passphraseFrom builds a config.PassphraseSource from an operator-chosen spec:
// "prompt" reads from the terminal without echo, "stdin" reads a line from in,
// "env:VAR" reads environment variable VAR, and "file:path" reads a file's
// contents. An empty spec yields a nil source (config assumed plaintext). A
// literal passphrase as the spec is intentionally unsupported — never pass the
// key as a command-line argument.
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

// promptPassphrase reads a passphrase from the controlling terminal without
// echo. It opens /dev/tty directly so it works even when stdin is the config
// pipe.
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

// requireServing enforces the ports a serving instance cannot run without: a
// signing identity to attest merges, storage to hold the graph, and at least one
// authenticator (noauth is a choice — there is no silent open-by-default). Run
// assembles only what is configured; this is where the serving policy lives.
func requireServing(app *config.App) error {
	var missing []string
	if app.Signer == nil {
		missing = append(missing, "signer")
	}
	if app.Storage == nil {
		missing = append(missing, "storage")
	}
	if len(app.Auth) == 0 {
		missing = append(missing, "auth")
	}
	if len(missing) > 0 {
		return fmt.Errorf("config is missing required section(s) for serving: %v", missing)
	}
	return nil
}

// serve runs the HTTP server until the process is signalled, then shuts it down
// gracefully. The surface is currently a health endpoint; the API mounts here
// once the endpoint adapter lands and --addr moves into the endpoints config.
func serve(addr string, app *config.App) error {
	logIdentity(app)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok\n")
	})
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() {
		slog.Info("ranke-db serving", "addr", addr)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		slog.Info("ranke-db shutting down")
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(sctx)
	}
}

// logIdentity reports the signing identity the stack assembled with, so the
// operator can confirm which key the server attests merges under.
func logIdentity(app *config.App) {
	pub := app.Signer.Public()
	id := fmt.Sprintf("%T", pub)
	if ed, ok := pub.(ed25519.PublicKey); ok {
		id = "ed25519:" + base64.RawStdEncoding.EncodeToString(ed)
	}
	slog.Info("ranke-db assembled", "signer", id)
}
