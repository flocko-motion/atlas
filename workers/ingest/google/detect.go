package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type takeoutType int

const (
	takeoutContacts takeoutType = iota
	takeoutCalendar
)

func (t takeoutType) String() string {
	switch t {
	case takeoutContacts:
		return "contacts"
	case takeoutCalendar:
		return "calendar"
	default:
		return "unknown"
	}
}

// detectTakeoutType looks at directory contents to determine the takeout type.
func detectTakeoutType(dir string) (takeoutType, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("read directory %s: %w", dir, err)
	}

	hasVCF := false
	hasICS := false

	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if e.IsDir() {
			// Check subdirectories for VCF/ICS files too
			subEntries, err := os.ReadDir(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			for _, se := range subEntries {
				subExt := strings.ToLower(filepath.Ext(se.Name()))
				if subExt == ".vcf" {
					hasVCF = true
				}
				if subExt == ".ics" {
					hasICS = true
				}
			}
		}
		if ext == ".vcf" {
			hasVCF = true
		}
		if ext == ".ics" {
			hasICS = true
		}
	}

	switch {
	case hasVCF:
		return takeoutContacts, nil
	case hasICS:
		return takeoutCalendar, nil
	default:
		return 0, fmt.Errorf("could not detect takeout type in %s (no .vcf or .ics files found)", dir)
	}
}
