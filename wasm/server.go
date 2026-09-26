package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"
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
		if err := writeAtomic(savePath(name), data); err != nil {
			log.Printf("save %s: %v", name, err)
			http.Error(w, "write error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// Schreibt erst in eine temporaere Datei im selben Verzeichnis und benennt sie dann
// um. Bricht der Vorgang ab - Container gestoppt, Platte voll -, bleibt der alte
// Spielstand heil, statt halb ueberschrieben liegen zu bleiben. Fuer das Umbenennen
// reicht Schreibrecht auf das Verzeichnis, auch wenn die alte Datei jemand anderem
// gehoert.
func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // nach dem Rename ein No-op
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
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

	// Ohne Timeouts haelt eine Verbindung, die nie fertig sendet, ihre Goroutine
	// beliebig lange. Die Grenzen sind grosszuegig, weil ein Spielstand bis 4 MB gross
	// sein kann und bmp.jsdos auch ueber eine langsame Leitung durchkommen soll.
	srv := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}

	log.Println("listening on :8080")
	log.Fatal(srv.ListenAndServe())
}
