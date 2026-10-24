package main

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const (
	publicDir   = "/public"
	maxSaveSize = 4 << 20 // 4 MB
)

// A variable so the tests can point it at a temporary directory.
var dataDir = "/data"

var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func savePath(name string) string {
	return filepath.Join(dataDir, name+".sav")
}

func handleSaves(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !validName.MatchString(name) {
		http.Error(w, "invalid name", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		http.ServeFile(w, r, savePath(name))

	case http.MethodPut:
		data, err := io.ReadAll(io.LimitReader(r.Body, maxSaveSize+1))
		if err != nil {
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}
		if len(data) > maxSaveSize {
			http.Error(w, "save too large", http.StatusRequestEntityTooLarge)
			return
		}
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		if err := writeAtomic(savePath(name), data); err != nil {
			slog.Error("save failed", "name", name, "err", err)
			http.Error(w, "write error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// Writes to a temporary file in the same directory first and then renames it. If
// the operation is interrupted - container stopped, disk full -, the old save stays
// intact instead of being left half overwritten. Renaming only needs write
// permission on the directory, even if the old file belongs to someone else.
func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // a no-op after the rename
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// These files change with every release and must therefore be revalidated on every
// request. Without that, the browser keeps caching them heuristically: the old
// bmp.jsdos stayed around, and because the autoexec sits IN the bundle, an outdated
// drive configuration without D: ran, even though the page itself was already
// current. no-cache does not forbid storing, it only requires revalidation; if the
// file is unchanged, the FileServer answers with 304 and nothing is transferred.
var immerNachfragen = map[string]bool{
	"/":            true,
	"/index.html":  true,
	"/bmp.jsdos":   true,
	"/version.txt": true,
	"/hilfe.html":  true,
}

func staticHandler() http.Handler {
	dateien := http.FileServer(http.Dir(publicDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /hilfe is the nicer address; the same file is served. The VNC variant has no
		// such shortcut because websockify serves the files there; that is why both pages
		// link to hilfe.html, which works everywhere.
		if r.URL.Path == "/hilfe" {
			r.URL.Path = "/hilfe.html"
		}
		if immerNachfragen[r.URL.Path] {
			w.Header().Set("Cache-Control", "no-cache")
		}
		dateien.ServeHTTP(w, r)
	})
}

func main() {
	// One JSON object per line, so `docker logs` can be filtered by field.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	http.Handle("/", staticHandler())
	http.HandleFunc("/api/saves/{name}", handleSaves)

	// Without timeouts, a connection that never finishes sending holds its goroutine
	// for as long as it likes. The limits are generous because a save can be up to
	// 4 MB and bmp.jsdos should get through over a slow line as well.
	srv := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}

	slog.Info("listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
