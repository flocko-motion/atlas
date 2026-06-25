// package: main / main
// type:    main
// job:     ranke-db entrypoint — wire the schemaf app: DB (JWT auth only), the REST endpoints, and an in-memory control plane (config + grants); archive persistence backends are chosen per-archive at runtime via the API
// limits:  wiring only. Control plane is throwaway in-memory (clean each run, test-friendly); a deployment wanting durable config/grants swaps those stores. vault + seal (/unlock) deferred.
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
	configmem "rankedb/adapter/config/mem"
	grantsmem "rankedb/adapter/grants/mem"
	"rankedb/api"
	"rankedb/assembler"
	"rankedb/core"
	"rankedb/db"
)

func main() {
	ctx := context.Background()
	app := schemaf.New(ctx)

	app.AddDb(db.Provider)        // schemaf DB — needed only for JWT auth (signing key in _schemaf_config)
	app.AddApi(api.Provider)      // register the REST endpoints
	app.SetFrontend(FrontendFS()) // embedded frontend assets (generated)

	roots := rootSubjects()

	// In-memory control plane (config + grants): throwaway, clean each run —
	// ideal for tests. ONLY an archive's own persistence (storage 𝒰 + sequencer
	// B_h) uses real backends, and those are chosen per-archive AT RUNTIME via
	// the create-archive API (so a test suite drives any backend). mem stores
	// need no DB, so we wire core synchronously here (race-free) before Run.
	// InternalDB (lazy) lets an archive opt into the server's own Postgres.
	authz := access.New(roots, grantsmem.New())
	c := core.New(authz, configmem.New(), assembler.Deps{InternalDB: func() *sql.DB { return schemafdb.DB() }})
	if err := c.Reconcile(ctx); err != nil { // empty config at boot → no archives
		log.Fatalf("core reconcile: %v", err)
	}
	api.Use(c)
	log.Printf("ranke-db ready (%d root subject(s); in-memory control plane)", len(roots))

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
