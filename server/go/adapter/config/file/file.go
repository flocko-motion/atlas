// package: file / adapter
// type:    adapter
// job:     load and save config from a file (yaml/json/toml/.env), flattening the parsed document to dotted-key Entries
// limits:  file-backed only — no env overlay (-> env), no remote/db store (-> postgres)
//
// Package file loads (and saves) the ranke-db server config from a file,
// auto-detecting the format by suffix: .yaml/.yml, .json, .toml, .env. The
// parsed document is flattened to config.Entries (dotted keys, e.g.
// "server.port"); nested maps become dotted paths and slices become indexed
// paths ("accounts.0.name"). A .env file's keys are used verbatim as config
// keys.
package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"

	"rankedb/adapter/config"
)

type format int

const (
	fmtYAML format = iota
	fmtJSON
	fmtTOML
	fmtEnv
)

// New returns a config Store backed by the file at path. The format is
// detected from the suffix; an unknown suffix is an error.
func New(path string) (config.Store, error) {
	f, err := detect(path)
	if err != nil {
		return nil, err
	}
	return &store{path: path, format: f}, nil
}

func detect(path string) (format, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return fmtYAML, nil
	case ".json":
		return fmtJSON, nil
	case ".toml":
		return fmtTOML, nil
	case ".env":
		return fmtEnv, nil
	default:
		return 0, fmt.Errorf("adapter/config/file: unsupported config format %q (want .yaml/.yml/.json/.toml/.env)", filepath.Ext(path))
	}
}

type store struct {
	path   string
	format format
}

func (s *store) Load(_ context.Context) (config.Entries, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("adapter/config/file: read %s: %w", s.path, err)
	}
	out := config.Entries{}
	if s.format == fmtEnv {
		return out, parseEnv(string(data), out)
	}
	var root any
	switch s.format {
	case fmtYAML:
		err = yaml.Unmarshal(data, &root)
	case fmtJSON:
		err = json.Unmarshal(data, &root)
	case fmtTOML:
		var m map[string]any
		err = toml.Unmarshal(data, &m)
		root = m
	}
	if err != nil {
		return nil, fmt.Errorf("adapter/config/file: parse %s: %w", s.path, err)
	}
	flatten("", root, out)
	return out, nil
}

func (s *store) Save(_ context.Context, e config.Entries) error {
	var (
		data []byte
		err  error
	)
	switch s.format {
	case fmtYAML:
		data, err = yaml.Marshal(map[string]string(e))
	case fmtJSON:
		data, err = json.MarshalIndent(map[string]string(e), "", "  ")
	case fmtTOML:
		var b strings.Builder
		err = toml.NewEncoder(&b).Encode(map[string]string(e))
		data = []byte(b.String())
	case fmtEnv:
		var b strings.Builder
		for k, v := range e {
			fmt.Fprintf(&b, "%s=%s\n", k, v)
		}
		data = []byte(b.String())
	}
	if err != nil {
		return fmt.Errorf("adapter/config/file: encode %s: %w", s.path, err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("adapter/config/file: write %s: %w", s.path, err)
	}
	return nil
}

func (s *store) Close() error { return nil }

// flatten turns a parsed document into dotted-key entries. Scalars are
// rendered with %v; nested maps/slices recurse with dotted/indexed paths.
func flatten(prefix string, v any, out config.Entries) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			flatten(join(prefix, k), val, out)
		}
	case map[any]any: // defensive: some decoders yield non-string keys
		for k, val := range t {
			flatten(join(prefix, fmt.Sprint(k)), val, out)
		}
	case []any:
		for i, val := range t {
			flatten(join(prefix, strconv.Itoa(i)), val, out)
		}
	default:
		if prefix != "" {
			out[prefix] = fmt.Sprint(t)
		}
	}
}

func join(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// parseEnv reads dotenv-style KEY=VALUE lines (keys used verbatim as config
// keys). Blank lines and # comments are skipped; surrounding quotes on the
// value are stripped.
func parseEnv(text string, out config.Entries) error {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		i := strings.IndexByte(line, '=')
		if i < 0 {
			return fmt.Errorf("adapter/config/file: malformed .env line %q (want KEY=VALUE)", line)
		}
		k := strings.TrimSpace(line[:i])
		v := strings.TrimSpace(line[i+1:])
		v = strings.Trim(v, `"'`)
		out[k] = v
	}
	return nil
}
