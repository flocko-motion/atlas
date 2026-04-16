package main

import (
	"context"
	"log"
	"rankedb/api"
	"rankedb/db"

	"github.com/flocko-motion/schemaf/schemaf"
)

func main() {
	ctx := context.Background()
	app := schemaf.New(ctx)

	app.AddDb(db.Provider)
	app.AddApi(api.Provider)
	app.SetFrontend(FrontendFS())

	if err := app.Run(); err != nil {
		log.Fatal(app.Run())
	}
}
