package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Routes the save API like main does, with the saves in a temporary directory.
func saveServer(t *testing.T) *httptest.Server {
	t.Helper()
	dataDir = t.TempDir()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/saves/{name}", handleSaves)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func request(t *testing.T, method, url string, body []byte) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	got, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, got
}

func TestSaveComesBackByteForByte(t *testing.T) {
	srv := saveServer(t)
	save := []byte{0, 1, 2, 0xff, '\n', 'B', 'M', 'P'}

	if code, _ := request(t, http.MethodPut, srv.URL+"/api/saves/slot_1", save); code != http.StatusNoContent {
		t.Fatalf("PUT: status %d, want %d", code, http.StatusNoContent)
	}
	code, got := request(t, http.MethodGet, srv.URL+"/api/saves/slot_1", nil)
	if code != http.StatusOK || !bytes.Equal(got, save) {
		t.Fatalf("GET: status %d, body %v, want %d and %v", code, got, http.StatusOK, save)
	}
}

func TestSaveOverwritesWithoutLeftovers(t *testing.T) {
	srv := saveServer(t)
	for _, save := range []string{"old and longer", "new"} {
		if code, _ := request(t, http.MethodPut, srv.URL+"/api/saves/s", []byte(save)); code != http.StatusNoContent {
			t.Fatalf("PUT %q: status %d", save, code)
		}
	}
	if _, got := request(t, http.MethodGet, srv.URL+"/api/saves/s", nil); string(got) != "new" {
		t.Errorf("GET: %q, want %q", got, "new")
	}
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "s.sav" {
		t.Errorf("data directory holds %v, want only s.sav", entries)
	}
}

func TestSaveRefusesInvalidNames(t *testing.T) {
	srv := saveServer(t)
	for _, name := range []string{"a.b", "a%20b", "%2e%2e", "%2e%2e%2fetc"} {
		if code, _ := request(t, http.MethodPut, srv.URL+"/api/saves/"+name, []byte("x")); code != http.StatusBadRequest {
			t.Errorf("PUT %q: status %d, want %d", name, code, http.StatusBadRequest)
		}
	}
}

func TestSaveRefusesOversizedSaves(t *testing.T) {
	srv := saveServer(t)
	save := []byte(strings.Repeat("x", maxSaveSize+1))
	if code, _ := request(t, http.MethodPut, srv.URL+"/api/saves/big", save); code != http.StatusRequestEntityTooLarge {
		t.Fatalf("PUT: status %d, want %d", code, http.StatusRequestEntityTooLarge)
	}
	if _, err := os.Stat(savePath("big")); !os.IsNotExist(err) {
		t.Errorf("oversized save was written: %v", err)
	}
}

func TestSaveRefusesOtherMethods(t *testing.T) {
	srv := saveServer(t)
	if code, _ := request(t, http.MethodDelete, srv.URL+"/api/saves/s", nil); code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE: status %d, want %d", code, http.StatusMethodNotAllowed)
	}
}

func TestMissingSaveIsNotFound(t *testing.T) {
	srv := saveServer(t)
	if code, _ := request(t, http.MethodGet, srv.URL+"/api/saves/none", nil); code != http.StatusNotFound {
		t.Errorf("GET: status %d, want %d", code, http.StatusNotFound)
	}
}
