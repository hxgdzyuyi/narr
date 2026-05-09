package model

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type VolumeCode struct {
	Prefix string
	Number int
}

type ChapterCode struct {
	VolumePrefix  string
	VolumeNumber  int
	ChapterPrefix string
	ChapterNumber int
}

func ParseVolumeCode(raw string) (VolumeCode, error) {
	prefix, number, err := parseCodeSegment(raw)
	if err != nil {
		return VolumeCode{}, fmt.Errorf("invalid volume code %q: %w", raw, err)
	}
	return VolumeCode{Prefix: prefix, Number: number}, nil
}

func (c VolumeCode) Canonical() string {
	return fmt.Sprintf("%s%02d", c.Prefix, c.Number)
}

func (c VolumeCode) Compare(other VolumeCode) int {
	if c.Prefix != other.Prefix {
		return strings.Compare(c.Prefix, other.Prefix)
	}
	switch {
	case c.Number < other.Number:
		return -1
	case c.Number > other.Number:
		return 1
	default:
		return 0
	}
}

func ParseChapterCode(raw string) (ChapterCode, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return ChapterCode{}, fmt.Errorf("chapter code must contain exactly one dot")
	}
	volumePrefix, volumeNumber, err := parseCodeSegment(parts[0])
	if err != nil {
		return ChapterCode{}, fmt.Errorf("invalid volume segment: %w", err)
	}
	chapterPrefix, chapterNumber, err := parseCodeSegment(parts[1])
	if err != nil {
		return ChapterCode{}, fmt.Errorf("invalid chapter segment: %w", err)
	}
	return ChapterCode{
		VolumePrefix:  volumePrefix,
		VolumeNumber:  volumeNumber,
		ChapterPrefix: chapterPrefix,
		ChapterNumber: chapterNumber,
	}, nil
}

func (c ChapterCode) Canonical() string {
	return fmt.Sprintf("%s%02d.%s%02d", c.VolumePrefix, c.VolumeNumber, c.ChapterPrefix, c.ChapterNumber)
}

func (c ChapterCode) VolumeCode() VolumeCode {
	return VolumeCode{Prefix: c.VolumePrefix, Number: c.VolumeNumber}
}

func (c ChapterCode) Compare(other ChapterCode) int {
	if c.VolumePrefix != other.VolumePrefix {
		return strings.Compare(c.VolumePrefix, other.VolumePrefix)
	}
	if c.ChapterPrefix != other.ChapterPrefix {
		return strings.Compare(c.ChapterPrefix, other.ChapterPrefix)
	}
	if c.VolumeNumber < other.VolumeNumber {
		return -1
	}
	if c.VolumeNumber > other.VolumeNumber {
		return 1
	}
	if c.ChapterNumber < other.ChapterNumber {
		return -1
	}
	if c.ChapterNumber > other.ChapterNumber {
		return 1
	}
	return 0
}

func parseCodeSegment(raw string) (string, int, error) {
	if raw == "" {
		return "", 0, fmt.Errorf("empty segment")
	}
	runes := []rune(raw)
	split := len(runes)
	for split > 0 && unicode.IsDigit(runes[split-1]) {
		split--
	}
	if split == len(runes) {
		return "", 0, fmt.Errorf("missing numeric suffix")
	}
	if split == 0 {
		return "", 0, fmt.Errorf("missing prefix")
	}
	prefix := string(runes[:split])
	for _, r := range prefix {
		if r == '_' || unicode.IsLetter(r) {
			continue
		}
		return "", 0, fmt.Errorf("invalid prefix character %q", r)
	}
	numberText := string(runes[split:])
	number, err := strconv.Atoi(numberText)
	if err != nil {
		return "", 0, err
	}
	return prefix, number, nil
}
