package model

import "testing"

func TestParseChapterCodeCanonicalizes(t *testing.T) {
	code, err := ParseChapterCode("vol1.ch001")
	if err != nil {
		t.Fatalf("ParseChapterCode returned error: %v", err)
	}
	if got := code.Canonical(); got != "vol01.ch01" {
		t.Fatalf("Canonical() = %q, want vol01.ch01", got)
	}
	if got := code.VolumeCode().Canonical(); got != "vol01" {
		t.Fatalf("VolumeCode().Canonical() = %q, want vol01", got)
	}
}

func TestParseChapterCodeRejectsInvalid(t *testing.T) {
	for _, raw := range []string{"vol.ch01", "vol01.ch", "vol01.ch01.x", "vol-01.ch01"} {
		if _, err := ParseChapterCode(raw); err == nil {
			t.Fatalf("ParseChapterCode(%q) returned no error", raw)
		}
	}
}
