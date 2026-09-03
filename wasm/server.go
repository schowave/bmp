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
	dataDir     = "/data"
	publicDir   = "/public"
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

// Diese Dateien aendern sich mit jedem Release und muessen deshalb bei jedem Aufruf
// nachgefragt werden. Ohne das cacht der Browser sie heuristisch weiter: die alte
// bmp.jsdos blieb liegen, und weil die autoexec IM Bundle steckt, lief eine veraltete
// Laufwerkskonfiguration ohne D:, obwohl die Seite selbst schon aktuell war. Ein
// no-cache verbietet nicht das Speichern, sondern verlangt nur die Rueckfrage; bei
// unveraenderter Datei antwortet der FileServer mit 304 und es wird nichts uebertragen.
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
		// /hilfe ist die schoenere Adresse, ausgeliefert wird dieselbe Datei. In der
		// VNC-Variante gibt es diese Abkuerzung nicht, weil dort websockify die Dateien
		// ausliefert; deshalb verlinken beide Seiten auf hilfe.html, was ueberall geht.
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
	http.Handle("/", staticHandler())
	http.HandleFunc("/api/saves/{name}", handleSaves)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
