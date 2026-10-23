package build

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/schowave/bmp/remake/internal/seasons"
)

// TestEmbeddedSeasonsMatchTheData builds the seasons from remake/data and
// compares them with those built into the remake.
func TestEmbeddedSeasonsMatchTheData(t *testing.T) {
	data := filepath.Join("..", "..", "..", "data")
	if _, err := os.Stat(filepath.Join(data, "clubs.csv")); err != nil {
		t.Skip("no raw data in", data)
	}
	out := t.TempDir()
	cfg := Config{Data: os.DirFS(data), Orig: os.DirFS(filepath.Join("..", "..", "embedded", "data")), Out: out, From: 1990, To: 2026, Log: t.Logf}
	if err := All(cfg); err != nil {
		t.Fatal(err)
	}
	want, got := os.DirFS(out), seasons.Files()
	n := 0
	err := fs.WalkDir(want, ".", func(path string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		a, _ := fs.ReadFile(want, path)
		b, err := fs.ReadFile(got, path)
		if err != nil || !bytes.Equal(a, b) {
			t.Errorf("%s differs from the build of remake/data, run mise run remake:seasons", path)
		}
		n++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	m := 0
	_ = fs.WalkDir(got, ".", func(path string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() && path != "README.md" {
			m++
		}
		return err
	})
	if m != n {
		t.Errorf("embedded %d files, the build has %d", m, n)
	}
}
