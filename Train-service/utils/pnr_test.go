package utils

import (
	"strings"
	"testing"
)

func TestGeneratePNR_Format(t *testing.T) {
	for i := 0; i < 1000; i++ {
		pnr, err := GeneratePNR()
		if err != nil {
			t.Fatalf("GeneratePNR returned error: %v", err)
		}

		// Must be exactly 6 characters
		if len(pnr) != PNRLength {
			t.Fatalf("expected PNR length %d, got %d: %q", PNRLength, len(pnr), pnr)
		}

		// Every character must be in the allowed charset
		for _, ch := range pnr {
			if !strings.ContainsRune(pnrCharset, ch) {
				t.Fatalf("PNR %q contains invalid character %q", pnr, ch)
			}
		}

		// Must not contain ambiguous characters
		for _, ambiguous := range []string{"0", "O", "I", "1"} {
			if strings.Contains(pnr, ambiguous) {
				t.Fatalf("PNR %q contains ambiguous character %q", pnr, ambiguous)
			}
		}
	}
}

func TestGeneratePNR_Uniqueness(t *testing.T) {
	const count = 10_000
	seen := make(map[string]bool, count)

	for i := 0; i < count; i++ {
		pnr, err := GeneratePNR()
		if err != nil {
			t.Fatalf("GeneratePNR error at iteration %d: %v", i, err)
		}
		if seen[pnr] {
			t.Fatalf("duplicate PNR generated: %q (at iteration %d)", pnr, i)
		}
		seen[pnr] = true
	}
}

func TestGeneratePNR_NotEmpty(t *testing.T) {
	pnr, err := GeneratePNR()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pnr == "" {
		t.Fatal("GeneratePNR returned empty string")
	}
}
