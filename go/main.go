package main

import (
	"context"
	"log"
	"os"
	"github.com/flocko-motion/rankedb/api"
	"github.com/flocko-motion/rankedb/db"
	"github.com/flocko-motion/rankedb/s3"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/flocko-motion/schemaf/schemaf"
)

func main() {
	ctx := context.Background()
	app := schemaf.New(ctx)

	app.AddDb(db.Provider)
	app.AddApi(api.Provider)
	app.SetFrontend(FrontendFS())

	// S3 init as a service — runs after compose services are healthy
	app.AddService(func(ctx context.Context) {
		if err := s3.Init(ctx); err != nil {
			log.Fatal("S3 startup check failed: ", err)
		}
	})

	// Register S3 health check — returns 503 if S3 unreachable
	schemafapi.RegisterHealth("s3", func() error {
		return s3.HealthCheck(context.Background())
	})

	// Register S3 status in /api/status
	schemafapi.RegisterStatus("s3", func() any {
		err := s3.HealthCheck(context.Background())
		return map[string]any{
			"bucket":    os.Getenv("S3_BUCKET"),
			"connected": err == nil,
		}
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
