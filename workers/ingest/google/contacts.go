package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rankedb "github.com/flocko-motion/rankedb-sdk"
)

// ingestContacts walks a Google Contacts takeout directory and ingests VCF + photos.
func ingestContacts(dir string, client *rankedb.Client, dryRun bool) error {
	// Discover all group directories (each has a .vcf + optional .jpg files)
	groups, err := discoverContactGroups(dir)
	if err != nil {
		return err
	}

	if len(groups) == 0 {
		return fmt.Errorf("no contact groups found in %s", dir)
	}

	fmt.Printf("Found %d contact group(s)\n", len(groups))

	if dryRun {
		return dryRunContacts(groups)
	}

	// Register worker + run
	configID, runID, err := client.StartRun(rankedb.WorkerConfig{
		Name:    "ranke-ingest-google",
		Version: "0.1.0",
		Params:  map[string]string{"type": "contacts"},
	})
	if err != nil {
		return fmt.Errorf("start run: %w", err)
	}
	fmt.Printf("Worker config: %s\nRun: %s\n\n", configID, runID)

	stats := struct {
		vcards, photos, skipped int
	}{}

	// Use "Alle Kontakte" as the primary source if it exists — it's the superset.
	// Other groups are just category labels, already embedded in each VCF's CATEGORIES field.
	primaryGroup := findPrimaryGroup(groups)
	if primaryGroup == nil {
		return fmt.Errorf("no primary contact group found (expected 'Alle Kontakte' or 'All Contacts' or a single .vcf)")
	}

	fmt.Printf("Primary group: %s (%d contacts)\n\n", primaryGroup.name, len(primaryGroup.cards))

	for _, card := range primaryGroup.cards {
		if card.FullName == "" && len(card.Raw) < 50 {
			stats.skipped++
			continue
		}

		// Ingest vCard as L0 source/data node
		origin := fmt.Sprintf("google-takeout/contacts/%s", primaryGroup.name)
		originalName := card.FullName + ".vcf"
		if card.FullName == "" {
			originalName = "unnamed.vcf"
		}

		_, err := client.CreateNode(rankedb.CreateNodeRequest{
			Level:          0,
			ContentClass:   "source",
			ContentType:    "data",
			EncodingClass:  "text",
			EncodingFormat: "vcard",
			Content:        &card.Raw,
			Origin:         rankedb.Ptr(origin),
			OriginalName:   rankedb.Ptr(originalName),
		})
		if err != nil {
			fmt.Printf("  WARN: failed to ingest vcard %q: %v\n", card.FullName, err)
			continue
		}
		stats.vcards++

		// Check for matching photo
		photoPath := findContactPhoto(dir, card.FullName)
		if photoPath != "" {
			photoData, err := os.ReadFile(photoPath)
			if err != nil {
				fmt.Printf("  WARN: failed to read photo %s: %v\n", photoPath, err)
			} else {
				photoContent := string(photoData)
				photoOrigin := fmt.Sprintf("google-takeout/contacts/%s/photo", primaryGroup.name)
				_, err := client.CreateNode(rankedb.CreateNodeRequest{
					Level:          0,
					ContentClass:   "source",
					ContentType:    "media",
					EncodingClass:  "image",
					EncodingFormat: "jpeg",
					Content:        &photoContent,
					Origin:         rankedb.Ptr(photoOrigin),
					OriginalName:   rankedb.Ptr(filepath.Base(photoPath)),
				})
				if err != nil {
					fmt.Printf("  WARN: failed to ingest photo for %q: %v\n", card.FullName, err)
				} else {
					stats.photos++
				}
			}
		}
	}

	fmt.Printf("\nDone. Ingested %d vCards, %d photos, skipped %d empty entries.\n", stats.vcards, stats.photos, stats.skipped)
	return nil
}

// contactGroup is a directory containing a VCF file and optional photos.
type contactGroup struct {
	name  string
	dir   string
	vcf   string // path to .vcf file
	cards []VCard
}

func discoverContactGroups(dir string) ([]contactGroup, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	var groups []contactGroup

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		subdir := filepath.Join(dir, e.Name())
		subEntries, err := os.ReadDir(subdir)
		if err != nil {
			continue
		}

		for _, se := range subEntries {
			if strings.ToLower(filepath.Ext(se.Name())) == ".vcf" {
				vcfPath := filepath.Join(subdir, se.Name())
				cards, err := ParseVCardFile(vcfPath)
				if err != nil {
					fmt.Printf("WARN: failed to parse %s: %v\n", vcfPath, err)
					continue
				}
				groups = append(groups, contactGroup{
					name:  e.Name(),
					dir:   subdir,
					vcf:   vcfPath,
					cards: cards,
				})
				break // one VCF per group directory
			}
		}
	}

	return groups, nil
}

func findPrimaryGroup(groups []contactGroup) *contactGroup {
	// Prefer "Alle Kontakte" (German) or "All Contacts" (English)
	for i := range groups {
		lower := strings.ToLower(groups[i].name)
		if lower == "alle kontakte" || lower == "all contacts" {
			return &groups[i]
		}
	}
	// Fall back to "Meine Kontakte" / "My Contacts"
	for i := range groups {
		lower := strings.ToLower(groups[i].name)
		if lower == "meine kontakte" || lower == "my contacts" {
			return &groups[i]
		}
	}
	// Fall back to the group with the most contacts
	if len(groups) > 0 {
		best := 0
		for i := range groups {
			if len(groups[i].cards) > len(groups[best].cards) {
				best = i
			}
		}
		return &groups[best]
	}
	return nil
}

// findContactPhoto searches for a JPG matching the contact's full name.
// Photos can be in the same group dir or any sibling group dir.
func findContactPhoto(baseDir string, fullName string) string {
	if fullName == "" {
		return ""
	}

	// Google Takeout names photos after the display name
	target := fullName + ".jpg"

	// Walk all subdirectories looking for the photo
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return ""
	}

	for _, e := range entries {
		if !e.IsDir() {
			// Check root dir too
			if e.Name() == target {
				return filepath.Join(baseDir, e.Name())
			}
			continue
		}
		candidate := filepath.Join(baseDir, e.Name(), target)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func dryRunContacts(groups []contactGroup) error {
	for _, g := range groups {
		fmt.Printf("  %s: %d contacts\n", g.name, len(g.cards))
	}

	primary := findPrimaryGroup(groups)
	if primary == nil {
		fmt.Println("\nNo primary group detected.")
		return nil
	}

	fmt.Printf("\nPrimary group: %s\n", primary.name)
	fmt.Printf("Would ingest %d vCard entries.\n\n", len(primary.cards))

	// Show sample
	limit := 10
	if len(primary.cards) < limit {
		limit = len(primary.cards)
	}
	fmt.Printf("First %d contacts:\n", limit)
	for i := 0; i < limit; i++ {
		card := primary.cards[i]
		name := card.FullName
		if name == "" {
			name = "(unnamed)"
		}
		cats := ""
		if len(card.Categories) > 0 {
			cats = " [" + strings.Join(card.Categories, ", ") + "]"
		}
		fmt.Printf("  %3d. %s%s\n", i+1, name, cats)
	}
	if len(primary.cards) > limit {
		fmt.Printf("  ... and %d more\n", len(primary.cards)-limit)
	}

	return nil
}
