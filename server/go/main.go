// package: main / main
// type:    main
// job:     ranke-db entrypoint — wire the schemaf app: DB + JWT auth, the REST endpoints, and the core (config → archives → lifecycle) bootstrapped once the DB is up
// limits:  wiring only. Config + grants live in the internal Postgres (server-operational data). DEFERRED, noted inline: the config-seed that defines tenants/archives, vault + seal (/unlock), and a readiness gate for the init race.
package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"

	schemafdb "github.com/flocko-motion/schemaf/db"
	"github.com/flocko-motion/schemaf/schemaf"

	"rankedb/access"
	configpg "rankedb/adapter/config/postgres"
	grantspg "rankedb/adapter/grants/postgres"
	"rankedb/api"
	"rankedb/assembler"
	"rankedb/core"
	"rankedb/db"
)

func main() {
	ctx := context.Background()
	app := schemaf.New(ctx)

	app.AddDb(db.Provider)        // DB connect + migrations + JWT auth (sets hasDB)
	app.AddApi(api.Provider)      // register the REST endpoints
	app.SetFrontend(FrontendFS()) // embedded frontend assets (generated)

	roots := rootSubjects()

	// Bootstrap the core AFTER the DB is up (AddService runs post-migration).
	// Server-operational data — grants (rights) and config — lives in the
	// internal Postgres; an archive's own 𝒰/sequencer backends are per-archive,
	// chosen by its config (-> assembler). NOTE: a request arriving before this
	// finishes would hit a nil core — acceptable for now; a readiness gate is a
	// follow-up. A failed store init is fatal (the server can't run without it).
	app.AddService(func(ctx context.Context) {
		conn := func() *sql.DB { return schemafdb.DB() }
		grantStore, err := grantspg.NewWithConn(ctx, conn)
		if err != nil {
			log.Fatalf("bootstrap: grant store: %v", err)
		}
		configStore, err := configpg.NewWithConn(ctx, conn)
		if err != nil {
			log.Fatalf("bootstrap: config store: %v", err)
		}
		authz := access.New(roots, grantStore)
		c := core.New(authz, configStore, assembler.Deps{InternalDB: conn})
		if err := c.Reconcile(ctx); err != nil {
			// Per-archive assembly failures are already isolated (Failed state);
			// a whole-config error (e.g. a bad name) lands here.
			log.Printf("bootstrap: reconcile: %v", err)
		}
		api.Use(c)
		log.Printf("ranke-db core ready (%d root subject(s) configured)", len(roots))
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// rootSubjects reads the break-glass root subjects from RANKE_ROOT_SUBJECT
// (comma-separated). It only NAMES root; acting as root still requires a valid
// JWT for that subject.
func rootSubjects() []string {
	var out []string
	for _, s := range strings.Split(os.Getenv("RANKE_ROOT_SUBJECT"), ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
