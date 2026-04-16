package main

import (
	"context"
	"fmt"
	"log"

	rankedb "github.com/flocko-motion/rankedb/worker"
)

func main() {
	ctx := context.Background()
	client := rankedb.NewClient("http://localhost:8000")

	// 1. Register worker config
	configID, err := client.CreateWorkerConfig(ctx, rankedb.WorkerConfig{
		Name:    "dummy-worker",
		Version: "0.1",
	})
	if err != nil {
		log.Fatal("create config: ", err)
	}
	fmt.Println("Config node:", configID)

	// 2. Start a run
	runID, err := client.StartRun(ctx, configID)
	if err != nil {
		log.Fatal("start run: ", err)
	}
	fmt.Println("Run ID:", runID)

	// 3. Check health
	fmt.Println("Worker ready. Server is reachable.")
}
