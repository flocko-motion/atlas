// package: assembler / spec-config
// type:    logic
// job:     read one archive's build Spec from a flat dotted-key config map (the config→Spec bridge)
// limits:  takes a plain map so the assembler stays config-agnostic; does not enumerate archives or resolve vault refs (-> core)
//
// Both storage.backend and sequencer.backend are REQUIRED — there is no silent
// default. The sequencer especially: B_h (its stored head) is the KEY to the
// archive — without it the content-addressed Universe has no entry point and
// cannot be loaded, and advancing it is the mint authority. So where B_h lives
// must be a deliberate, configured choice, never assumed.
package assembler

import (
	"fmt"
	"strings"
)

// SpecFromConfig reads one archive's build Spec from a flat, dotted-key config
// map under "archives.<name>.…". It is the bridge the server core uses to turn
// stored config into an assembler Spec; Assemble itself stays config-agnostic
// (this takes a plain map, not a config type).
//
// Keys:
//
//	archives.<name>.storage.backend    = mem | fs | sqlite
//	archives.<name>.storage.dir        = <path>   (fs)
//	archives.<name>.storage.dsn        = <dsn>    (sqlite)
//	archives.<name>.sequencer.backend  = mem | file | postgres
//	archives.<name>.sequencer.path     = <path>   (file)
//	archives.<name>.sequencer.dsn      = <dsn>    (postgres)
//	archives.<name>.sequencer.key      = <key>    (postgres; defaults to <name>)
func SpecFromConfig(entries map[string]string, name string) (Spec, error) {
	if name == "" {
		return Spec{}, fmt.Errorf("assembler: empty archive name")
	}
	p := "archives." + name + "."
	get := func(k string) string { return strings.TrimSpace(entries[p+k]) }

	storage := StorageSpec{
		Backend: get("storage.backend"),
		Dir:     get("storage.dir"),
		DSN:     get("storage.dsn"),
	}
	if storage.Backend == "" {
		return Spec{}, fmt.Errorf("assembler: archive %q: missing storage.backend", name)
	}

	seq := SequencerSpec{
		Backend: get("sequencer.backend"),
		Path:    get("sequencer.path"),
		DSN:     get("sequencer.dsn"),
		Key:     get("sequencer.key"),
	}
	if seq.Backend == "" {
		return Spec{}, fmt.Errorf("assembler: archive %q: missing sequencer.backend — B_h is the archive's key (the Universe is unloadable without it); choose where it is stored explicitly", name)
	}
	if seq.Backend == "postgres" && seq.Key == "" {
		seq.Key = name // sensible default: the archive name keys its head row
	}

	return Spec{Storage: storage, Sequencer: seq}, nil
}
