package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// newTestDatasetServer arranca un servidor TLS local y devuelve el
// cliente y el allowlist (su propio host) que un DatasetSpec de
// prueba puede usar sin tocar la red real.
func newTestDatasetServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *http.Client, []string) {
	t.Helper()

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	parsedURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("no se pudo parsear la URL del servidor de prueba: %v", err)
	}

	return server, server.Client(), []string{parsedURL.Hostname()}
}

func testPayloadSpec(t *testing.T, name string, payload []byte, url string) DatasetSpec {
	t.Helper()

	sum := sha256.Sum256(payload)

	return DatasetSpec{
		Name:   name,
		URL:    url,
		Path:   filepath.Join(t.TempDir(), name+".csv"),
		Size:   int64(len(payload)),
		SHA256: hex.EncodeToString(sum[:]),
	}
}

func TestEnsureDataset_HappyPathDownloads(t *testing.T) {
	payload := bytes.Repeat([]byte("A"), 4096)
	hits := 0

	server, client, allowlist := newTestDatasetServer(t, func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(payload)
	})

	spec := testPayloadSpec(t, "happy", payload, server.URL)

	downloaded, err := ensureDataset(context.Background(), client, spec, allowlist, os.Stdout)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if !downloaded {
		t.Error("downloaded = false, se esperaba true")
	}

	if hits != 1 {
		t.Errorf("hits = %d, se esperaba 1", hits)
	}

	content, err := os.ReadFile(spec.Path)
	if err != nil {
		t.Fatalf("no se pudo leer el archivo final: %v", err)
	}

	if !bytes.Equal(content, payload) {
		t.Error("el contenido descargado no coincide con el payload")
	}

	if _, err := os.Stat(spec.Path + ".part"); !os.IsNotExist(err) {
		t.Errorf("se esperaba que no quedara .part, stat err = %v", err)
	}
}

func TestEnsureDataset_HashMismatch(t *testing.T) {
	payload := bytes.Repeat([]byte("B"), 2048)

	server, client, allowlist := newTestDatasetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(payload)
	})

	spec := testPayloadSpec(t, "badhash", payload, server.URL)
	spec.SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"[:64]

	_, err := ensureDataset(context.Background(), client, spec, allowlist, os.Stdout)

	if err == nil {
		t.Fatal("se esperaba un error por hash inválido")
	}

	assertNoFinalOrPartFile(t, spec.Path)
}

func TestEnsureDataset_TruncatedBody(t *testing.T) {
	fullPayload := bytes.Repeat([]byte("C"), 8192)
	shortPayload := fullPayload[:4096]

	server, client, allowlist := newTestDatasetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		// Anuncia el tamaño completo pero corta la conexión antes de
		// terminar de escribirlo, simulando una descarga truncada.
		w.Header().Set("Content-Length", strconv.Itoa(len(fullPayload)))
		w.Write(shortPayload)
	})

	spec := testPayloadSpec(t, "truncated", fullPayload, server.URL)

	_, err := ensureDataset(context.Background(), client, spec, allowlist, os.Stdout)

	if err == nil {
		t.Fatal("se esperaba un error por descarga truncada")
	}

	assertNoFinalOrPartFile(t, spec.Path)
}

func TestEnsureDataset_ContentLengthMismatch(t *testing.T) {
	payload := bytes.Repeat([]byte("D"), 1024)

	server, client, allowlist := newTestDatasetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)+500))
		w.Write(payload)
	})

	spec := testPayloadSpec(t, "clmismatch", payload, server.URL)

	_, err := ensureDataset(context.Background(), client, spec, allowlist, os.Stdout)

	if err == nil {
		t.Fatal("se esperaba un error por Content-Length distinto al esperado")
	}

	assertNoFinalOrPartFile(t, spec.Path)
}

func TestEnsureDataset_RejectsHTMLContentType(t *testing.T) {
	payload := []byte("<html>cuota excedida</html>")

	server, client, allowlist := newTestDatasetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(payload)
	})

	spec := testPayloadSpec(t, "html", []byte("no importa, se rechaza antes"), server.URL)
	spec.Size = int64(len(payload))

	_, err := ensureDataset(context.Background(), client, spec, allowlist, os.Stdout)

	if err == nil {
		t.Fatal("se esperaba un error por Content-Type text/html")
	}

	assertNoFinalOrPartFile(t, spec.Path)
}

func TestEnsureDataset_RejectsNon200(t *testing.T) {
	server, client, allowlist := newTestDatasetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	spec := testPayloadSpec(t, "notfound", []byte("x"), server.URL)

	_, err := ensureDataset(context.Background(), client, spec, allowlist, os.Stdout)

	if err == nil {
		t.Fatal("se esperaba un error por status distinto de 200")
	}

	assertNoFinalOrPartFile(t, spec.Path)
}

func TestEnsureDataset_RejectsHTTPScheme(t *testing.T) {
	client := &http.Client{}
	spec := testPayloadSpec(t, "plainhttp", []byte("x"), "http://example.invalid/data")

	_, err := ensureDataset(context.Background(), client, spec, []string{"example.invalid"}, os.Stdout)

	if err == nil {
		t.Fatal("se esperaba un error por esquema http (no https)")
	}

	assertNoFinalOrPartFile(t, spec.Path)
}

func TestEnsureDataset_RejectsRedirectToDisallowedHost(t *testing.T) {
	server, client, allowlist := newTestDatasetServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.example.invalid/data", http.StatusFound)
	})

	spec := testPayloadSpec(t, "redirect", []byte("x"), server.URL)

	_, err := ensureDataset(context.Background(), client, spec, allowlist, os.Stdout)

	if err == nil {
		t.Fatal("se esperaba un error por redirección a un host no permitido")
	}

	assertNoFinalOrPartFile(t, spec.Path)
}

func TestEnsureDataset_SkipsExistingValidFile(t *testing.T) {
	payload := bytes.Repeat([]byte("E"), 1500)
	hits := 0

	server, client, allowlist := newTestDatasetServer(t, func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(payload)
	})

	spec := testPayloadSpec(t, "existingvalid", payload, server.URL)

	if err := os.WriteFile(spec.Path, payload, 0o644); err != nil {
		t.Fatalf("no se pudo preparar el archivo existente: %v", err)
	}

	downloaded, err := ensureDataset(context.Background(), client, spec, allowlist, os.Stdout)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if downloaded {
		t.Error("downloaded = true, se esperaba false (archivo ya válido)")
	}

	if hits != 0 {
		t.Errorf("hits = %d, se esperaba 0 (no debía descargar)", hits)
	}
}

func TestEnsureDataset_ReplacesExistingCorruptFile(t *testing.T) {
	payload := bytes.Repeat([]byte("F"), 3000)
	hits := 0

	server, client, allowlist := newTestDatasetServer(t, func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(payload)
	})

	spec := testPayloadSpec(t, "existingcorrupt", payload, server.URL)

	if err := os.WriteFile(spec.Path, []byte("contenido corrupto, no coincide"), 0o644); err != nil {
		t.Fatalf("no se pudo preparar el archivo corrupto: %v", err)
	}

	downloaded, err := ensureDataset(context.Background(), client, spec, allowlist, os.Stdout)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if !downloaded {
		t.Error("downloaded = false, se esperaba true (debía reemplazar el archivo corrupto)")
	}

	if hits != 1 {
		t.Errorf("hits = %d, se esperaba 1", hits)
	}

	content, err := os.ReadFile(spec.Path)
	if err != nil {
		t.Fatalf("no se pudo leer el archivo final: %v", err)
	}

	if !bytes.Equal(content, payload) {
		t.Error("el contenido final no coincide con el payload nuevo")
	}
}

func TestEnsureDataset_HonorsContextCancellation(t *testing.T) {
	server, client, allowlist := newTestDatasetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("x"))
	})

	spec := testPayloadSpec(t, "cancelled", []byte("x"), server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ensureDataset(ctx, client, spec, allowlist, os.Stdout)

	if err == nil {
		t.Fatal("se esperaba un error por contexto cancelado")
	}
}

func assertNoFinalOrPartFile(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("se esperaba que no existiera %s, stat err = %v", path, err)
	}

	if _, err := os.Stat(path + ".part"); !os.IsNotExist(err) {
		t.Errorf("se esperaba que no existiera %s.part, stat err = %v", path, err)
	}
}

func TestVerifyFile(t *testing.T) {
	dir := t.TempDir()
	payload := bytes.Repeat([]byte("G"), 777)
	sum := sha256.Sum256(payload)
	sha := hex.EncodeToString(sum[:])

	validPath := filepath.Join(dir, "valid.csv")
	if err := os.WriteFile(validPath, payload, 0o644); err != nil {
		t.Fatalf("no se pudo escribir el archivo de prueba: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		size    int64
		sha     string
		wantErr bool
	}{
		{"valido", validPath, int64(len(payload)), sha, false},
		{"tamaño incorrecto", validPath, int64(len(payload)) + 1, sha, true},
		{"hash incorrecto", validPath, int64(len(payload)), "0000000000000000000000000000000000000000000000000000000000000000"[:64], true},
		{"archivo inexistente", filepath.Join(dir, "no-existe.csv"), int64(len(payload)), sha, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyFile(tt.path, tt.size, tt.sha)

			if tt.wantErr && err == nil {
				t.Fatal("se esperaba un error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
		})
	}
}

func TestHostAllowed(t *testing.T) {
	allowlist := []string{
		"drive.usercontent.google.com",
		"drive.google.com",
		"*.googleusercontent.com",
	}

	tests := []struct {
		host string
		want bool
	}{
		{"drive.usercontent.google.com", true},
		{"drive.google.com", true},
		{"lh3.googleusercontent.com", true},
		{"googleusercontent.com", false},
		{"evil.example.com", false},
		{"drive.usercontent.google.com.evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := hostAllowed(tt.host, allowlist); got != tt.want {
				t.Errorf("hostAllowed(%q) = %v, se esperaba %v", tt.host, got, tt.want)
			}
		})
	}
}

func TestDatasetSpecsRegistry(t *testing.T) {
	clean, ok := datasetSpecs["clean"]
	if !ok {
		t.Fatal("se esperaba una entrada 'clean' en datasetSpecs")
	}

	if clean.Path != cleanDatasetPath {
		t.Errorf("clean.Path = %q, se esperaba %q", clean.Path, cleanDatasetPath)
	}

	if clean.Size != 244031060 {
		t.Errorf("clean.Size = %d, se esperaba 244031060", clean.Size)
	}

	raw, ok := datasetSpecs["raw"]
	if !ok {
		t.Fatal("se esperaba una entrada 'raw' en datasetSpecs")
	}

	if raw.Path != inputCSVPath {
		t.Errorf("raw.Path = %q, se esperaba %q", raw.Path, inputCSVPath)
	}

	if raw.Size != 947122735 {
		t.Errorf("raw.Size = %d, se esperaba 947122735", raw.Size)
	}
}

func TestEnsureDataset_IdleTimeoutWhileStreaming(t *testing.T) {
	previous := datasetIdleTimeout
	datasetIdleTimeout = 200 * time.Millisecond
	t.Cleanup(func() { datasetIdleTimeout = previous })

	server, client, allowlist := newTestDatasetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(bytes.Repeat([]byte("D"), 1024))
		w.(http.Flusher).Flush()
		// Stall: no more bytes until the client gives up.
		<-r.Context().Done()
	})

	spec := testPayloadSpec(t, "stalled", bytes.Repeat([]byte("D"), 8192), server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	_, err := ensureDataset(ctx, client, spec, allowlist, io.Discard)

	if err == nil {
		t.Fatal("se esperaba un error por inactividad durante la descarga")
	}

	if !strings.Contains(err.Error(), "inactividad") {
		t.Errorf("el error debería mencionar la inactividad, se obtuvo: %v", err)
	}

	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("la descarga tardó %v en fallar; el timeout de inactividad no se aplicó", elapsed)
	}

	if _, statErr := os.Stat(spec.Path); !os.IsNotExist(statErr) {
		t.Errorf("no debería existir el archivo final, stat err = %v", statErr)
	}
}
