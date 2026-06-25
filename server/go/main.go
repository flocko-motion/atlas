// package: main / main
// type:    main
// job:     schemaf app entrypoint — build the app and run it
// limits:  no business logic; wiring only (-> assembler, REST layer, /unlock to follow)
package main

import (
	"context"
	"log"

	"github.com/flocko-motion/schemaf/schemaf"
)

func main() {
	// The ranke graph now lives in the ranke-go library. The old in-Postgres
	// graph implementation (api/db/ranke-cli/apiclient/s3) has been purged.
	// Next to wire here: the config-driven assembler (config → Archive) and a
	// thin REST layer over ranke-go's Archive, plus the /unlock endpoint.
	app := schemaf.New(context.Background())
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
