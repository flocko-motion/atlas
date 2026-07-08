package storage

import "testing"

// TestParseSize covers the human-readable size suffixes the maxContentSize field
// accepts.
func TestParseSize(t *testing.T) {
	cases := map[string]uint64{
		"":      0,
		"512":   512,
		"8kb":   8 << 10,
		"2 MB":  2 << 20,
		"1gb":   1 << 30,
		"4096b": 4096,
	}
	for in, want := range cases {
		got, err := parseSize(in)
		if err != nil {
			t.Errorf("parseSize(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseSize(%q) = %d, want %d", in, got, want)
		}
	}
	if _, err := parseSize("not-a-size"); err == nil {
		t.Error("parseSize(\"not-a-size\") = nil error, want error")
	}
}
