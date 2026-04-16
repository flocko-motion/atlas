package main

import (
	"context"
	"log"

	"github.com/flocko-motion/schemaf/schemaf"

	"rankedb/api"
	"rankedb/db"
)

func main() {
	ctx := context.Background()
	app := schemaf.New(ctx)

	app.AddDb(db.Provider)
	app.AddApi(api.Provider)
	app.SetFrontend(FrontendFS())

	log.Fatal(app.Run())
}
