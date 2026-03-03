package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

const (
	dataDir    = "/data"
	publicDir  = "/public"
	maxSaveSize = 4 << 20 // 4 MB
)

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
		if err := os.WriteFile(savePath(name), data, 0644); err != nil {
			http.Error(w, "write error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	http.Handle("/", http.FileServer(http.Dir(publicDir)))
	http.HandleFunc("/api/saves/{name}", handleSaves)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
