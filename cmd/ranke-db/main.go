// package: main / cmd
// type:    entrypoint
// job:     the ranke-db binary — "run <configfile>" loads, decrypts, resolves, assembles, and serves
// limits:  CLI wiring only; loading/age lives in config, serving lifecycle here (-> config)
//
// Command ranke-db is the single binary. "run <configfile>|-" reads the launch
// artifact from a file or stdin, decrypts it if it is age-encrypted (passphrase
// from --age-key: prompt|stdin|env:VAR|file:path), validates and resolves it,
// assembles the adapter stack, and serves until SIGINT/SIGTERM. Authoring
// subcommands (config, tui) and the full API surface land in later passes.
package main

import (
	"bytes"
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
	"syscall"
	"time"

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
	case "-h", "--help", "help":
		return usage()
	default:
		return fmt.Errorf("unknown command %q (try: ranke-db run <configfile>)", args[0])
	}
}

func usage() error {
	fmt.Fprint(os.Stderr, `ranke-db — serve a Ranke-Graph from a launch artifact

usage:
  ranke-db run [--addr ADDR] [--age-key SRC] <configfile>|-

  <configfile>|-   path to the JSON launch artifact, or - to read from stdin
  --addr ADDR      address to serve on (default :8080)
  --age-key SRC    passphrase source for an age-encrypted config:
                   prompt | stdin | env:VAR | file:path
`)
	return nil
}

// cmdRun implements "ranke-db run".
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
	path := fs.Arg(0)

	fromStdin := path == "-"
	if fromStdin && *ageKey == "stdin" {
		return errors.New("cannot read both the config and the age key from stdin; use --age-key prompt|env:VAR|file:path")
	}

	var data []byte
	var err error
	if fromStdin {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	src, err := config.PassphraseFrom(*ageKey, os.Stdin)
	if err != nil {
		return err
	}
	plaintext, err := config.Decrypt(data, src)
	if err != nil {
		return err
	}

	c, err := config.Load(bytes.NewReader(plaintext))
	if err != nil {
		return err
	}

	ctx := context.Background()
	v, err := c.BuildVault(ctx)
	if err != nil {
		return err
	}
	app, err := c.Build(ctx, v)
	if err != nil {
		return err
	}
	if err := requireServing(app); err != nil {
		return err
	}
	return serve(*addr, app)
}

// requireServing enforces the ports a serving instance cannot run without: a
// signing identity to attest merges, storage to hold the graph, and an explicit
// auth choice (noauth is a choice — there is no silent open-by-default). Build
// assembles only what is configured; this is where the serving policy lives.
func requireServing(app *config.App) error {
	var missing []string
	if app.Signer == nil {
		missing = append(missing, "signer")
	}
	if app.Storage == nil {
		missing = append(missing, "storage")
	}
	if app.Auth == nil {
		missing = append(missing, "auth")
	}
	if len(missing) > 0 {
		return fmt.Errorf("config is missing required section(s) for serving: %v", missing)
	}
	return nil
}

// serve runs the HTTP server until the process is signalled, then shuts it down
// gracefully. The surface is currently a health endpoint; the API mounts here
// once the endpoints adapter lands.
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
