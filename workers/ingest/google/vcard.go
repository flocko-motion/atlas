package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// VCard represents a single parsed vCard entry.
type VCard struct {
	Raw        string // the full vCard text (BEGIN:VCARD ... END:VCARD)
	FullName   string // FN field
	Categories []string
}

// ParseVCardFile reads a .vcf file and returns individual vCard entries.
func ParseVCardFile(path string) ([]VCard, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var cards []VCard
	var lines []string
	inCard := false

	scanner := bufio.NewScanner(f)
	// VCF files can have long lines (PHOTO URLs with line folding)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// vCard line folding: a line starting with space/tab is a continuation
		if len(lines) > 0 && len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			lines[len(lines)-1] += line[1:]
			continue
		}

		if strings.TrimSpace(strings.ToUpper(line)) == "BEGIN:VCARD" {
			inCard = true
			lines = []string{line}
			continue
		}

		if !inCard {
			continue
		}

		lines = append(lines, line)

		if strings.TrimSpace(strings.ToUpper(line)) == "END:VCARD" {
			card := buildVCard(lines)
			cards = append(cards, card)
			inCard = false
			lines = nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return cards, nil
}

func buildVCard(lines []string) VCard {
	card := VCard{
		Raw: strings.Join(lines, "\r\n"),
	}

	for _, line := range lines {
		upper := strings.ToUpper(line)

		if strings.HasPrefix(upper, "FN:") || strings.HasPrefix(upper, "FN;") {
			// FN can have parameters like FN;CHARSET=UTF-8:Name
			idx := strings.Index(line, ":")
			if idx >= 0 {
				card.FullName = strings.TrimSpace(line[idx+1:])
			}
		}

		if strings.HasPrefix(upper, "CATEGORIES:") {
			idx := strings.Index(line, ":")
			if idx >= 0 {
				cats := strings.Split(line[idx+1:], ",")
				for _, c := range cats {
					c = strings.TrimSpace(c)
					if c != "" {
						card.Categories = append(card.Categories, c)
					}
				}
			}
		}
	}

	return card
}
