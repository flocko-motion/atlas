package main

import (
	"fmt"

	rankedb "github.com/flocko-motion/rankedb-sdk"
)

func run(dir string, server string, dryRun bool) error {
	detected, err := detectTakeoutType(dir)
	if err != nil {
		return err
	}

	fmt.Printf("Detected takeout type: %s\n", detected)
	if dryRun {
		fmt.Println("(dry run — no data will be sent to server)\n")
	} else {
		fmt.Printf("Server: %s\n\n", server)
	}

	var client *rankedb.Client
	if !dryRun {
		client = rankedb.NewClient(server)
	}

	switch detected {
	case takeoutContacts:
		return ingestContacts(dir, client, dryRun)
	case takeoutCalendar:
		return fmt.Errorf("calendar ingestion not yet implemented")
	default:
		return fmt.Errorf("unsupported takeout type: %s", detected)
	}
}
