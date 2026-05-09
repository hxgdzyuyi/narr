package source

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestErrorCodesAreDocumented(t *testing.T) {
	root := filepath.Join("..", "..")
	docs, err := os.ReadFile(filepath.Join(root, "docs", "error-codes.md"))
	if err != nil {
		t.Fatalf("failed to read docs/error-codes.md: %v", err)
	}
	documented := string(docs)
	used := map[string]bool{}
	codePattern := regexp.MustCompile(`"[EW][0-9]{4}"`)
	err = filepath.WalkDir(filepath.Join(root, "internal"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range codePattern.FindAllString(string(data), -1) {
			used[strings.Trim(match, `"`)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to scan Go files: %v", err)
	}
	for code := range used {
		if !strings.Contains(documented, "`"+code+"`") {
			t.Fatalf("%s is used in Go code but missing from docs/error-codes.md", code)
		}
	}
}
