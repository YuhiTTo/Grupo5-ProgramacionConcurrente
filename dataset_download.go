package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// DatasetSpec describe un dataset descargable: de dónde traerlo, dónde
// colocarlo y con qué tamaño/hash verificar la integridad del archivo.
type DatasetSpec struct {
	Name   string
	URL    string
	Path   string
	Size   int64
	SHA256 string
}

const (
	cleanDatasetDriveID = "1cQwAvdyhqbVbZN5Wj8cJuv3X1kOPbJoh"
	cleanDatasetSize    = 244031060
	cleanDatasetSHA256  = "cf2d180c8732f1f4e89a2feca08567e70d7b4b74ed045add7b72557121078e3c"

	rawDatasetDriveID = "1Tf4ubJF3OuRpqYnGQBM-nQEQsxjlvFGw"
	rawDatasetSize    = 947122735
	rawDatasetSHA256  = "52df97e39003b95bff4add93b4be2447554f1f249558a08ae650209730dad81f"

	// maxDatasetRedirects limita cuántas redirecciones se siguen antes
	// de abortar la descarga.
	maxDatasetRedirects = 10
)

// datasetIdleTimeout es el tiempo máximo sin recibir bytes mientras se
// transmite el cuerpo de la descarga. Es una variable (no constante)
// para que las pruebas puedan reducirlo.
var datasetIdleTimeout = 60 * time.Second

// defaultDatasetAllowlist son los únicos hosts desde los que se acepta
// descargar un dataset (Google Drive y su CDN de contenido).
var defaultDatasetAllowlist = []string{
	"drive.usercontent.google.com",
	"drive.google.com",
	"*.googleusercontent.com",
}

// datasetSpecs es el registro de datasets conocidos por el CLI.
var datasetSpecs = map[string]DatasetSpec{
	"clean": {
		Name:   "clean",
		URL:    driveDownloadURL(cleanDatasetDriveID),
		Path:   cleanDatasetPath,
		Size:   cleanDatasetSize,
		SHA256: cleanDatasetSHA256,
	},
	"raw": {
		Name:   "raw",
		URL:    driveDownloadURL(rawDatasetDriveID),
		Path:   inputCSVPath,
		Size:   rawDatasetSize,
		SHA256: rawDatasetSHA256,
	},
}

// driveDownloadURL construye la URL directa de descarga de Google Drive
// que evita la página de confirmación intermedia.
func driveDownloadURL(driveID string) string {
	return fmt.Sprintf(
		"https://drive.usercontent.google.com/download?id=%s&export=download&confirm=t",
		driveID,
	)
}

// newDatasetHTTPClient crea un cliente HTTP apto para descargar
// archivos grandes: tiene timeouts en las fases de conexión (dial, TLS
// handshake, cabeceras de respuesta) pero deliberadamente NO tiene un
// Client.Timeout global, que cortaría descargas largas y legítimas.
func newDatasetHTTPClient() *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	return &http.Client{Transport: transport}
}

// ensureDataset verifica si el dataset descrito por spec ya existe y es
// válido en disco; si no, lo descarga de forma segura. allowlist
// restringe los hosts aceptados (parámetro, no global, para que las
// pruebas puedan usar un servidor httptest local).
func ensureDataset(
	ctx context.Context,
	client *http.Client,
	spec DatasetSpec,
	allowlist []string,
	out io.Writer,
) (downloaded bool, err error) {

	if _, statErr := os.Stat(spec.Path); statErr == nil {
		verifyErr := verifyFile(spec.Path, spec.Size, spec.SHA256)

		if verifyErr == nil {
			fmt.Fprintf(out, "%s: %s ya existe y está verificado\n", spec.Name, spec.Path)
			return false, nil
		}

		fmt.Fprintf(
			out,
			"%s: el archivo existente en %s no es válido (%v); se volverá a descargar\n",
			spec.Name, spec.Path, verifyErr,
		)
	}

	if err := downloadDataset(ctx, client, spec, allowlist, out); err != nil {
		return false, err
	}

	return true, nil
}

// verifyFile comprueba que el archivo en path tenga exactamente size
// bytes y que su SHA-256 coincida con sha (comparación insensible a
// mayúsculas/minúsculas).
func verifyFile(path string, size int64, sha string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("no se pudo acceder al archivo: %w", err)
	}

	if info.Size() != size {
		return fmt.Errorf(
			"tamaño %d no coincide con el esperado %d",
			info.Size(), size,
		)
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()

	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("no se pudo calcular el SHA-256: %w", err)
	}

	computed := hex.EncodeToString(hasher.Sum(nil))

	if !strings.EqualFold(computed, sha) {
		return fmt.Errorf(
			"SHA-256 %s no coincide con el esperado %s",
			computed, sha,
		)
	}

	return nil
}

// downloadDataset hace la petición HTTP, valida la respuesta y
// transmite el cuerpo a disco de forma atómica.
func downloadDataset(
	ctx context.Context,
	client *http.Client,
	spec DatasetSpec,
	allowlist []string,
	out io.Writer,
) error {

	if err := validateDatasetURL(spec.URL, allowlist); err != nil {
		return fmt.Errorf("%s: %w", spec.Name, err)
	}

	// El contexto derivado permite cancelar la lectura del cuerpo cuando
	// el servidor deja de enviar datos (ver idleTimeoutReader).
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spec.URL, nil)
	if err != nil {
		return fmt.Errorf("%s: no se pudo construir la petición: %w", spec.Name, err)
	}

	requestClient := *client
	requestClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxDatasetRedirects {
			return fmt.Errorf("demasiadas redirecciones (%d)", len(via))
		}

		return validateDatasetURL(req.URL.String(), allowlist)
	}

	resp, err := requestClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: la descarga falló: %w", spec.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: respuesta HTTP inesperada: %s", spec.Name, resp.Status)
	}

	if mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); strings.EqualFold(mediaType, "text/html") {
		return fmt.Errorf(
			"%s: Google Drive devolvió una página HTML (posible aviso o cuota excedida); "+
				"use el enlace manual del README del dataset",
			spec.Name,
		)
	}

	if resp.ContentLength >= 0 && resp.ContentLength != spec.Size {
		return fmt.Errorf(
			"%s: el tamaño anunciado por el servidor (%d) no coincide con el esperado (%d)",
			spec.Name, resp.ContentLength, spec.Size,
		)
	}

	idleBody := newIdleTimeoutReader(resp.Body, datasetIdleTimeout, cancel)

	return streamToFile(idleBody, spec, out)
}

// idleTimeoutReader envuelve un io.Reader y cancela la operación (vía
// cancel) si pasan más de timeout sin recibir ningún byte. Cada lectura
// que devuelve datos reinicia el temporizador.
type idleTimeoutReader struct {
	reader   io.Reader
	timer    *time.Timer
	timeout  time.Duration
	timedOut atomic.Bool
}

func newIdleTimeoutReader(reader io.Reader, timeout time.Duration, cancel context.CancelFunc) *idleTimeoutReader {
	r := &idleTimeoutReader{reader: reader, timeout: timeout}

	r.timer = time.AfterFunc(timeout, func() {
		r.timedOut.Store(true)
		cancel()
	})

	return r
}

func (r *idleTimeoutReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)

	if err == io.EOF {
		r.timer.Stop()
		return n, err
	}

	if r.timedOut.Load() {
		r.timer.Stop()
		return n, fmt.Errorf("sin recibir datos durante %v (timeout de inactividad)", r.timeout)
	}

	if err != nil {
		r.timer.Stop()
		return n, err
	}

	if n > 0 {
		r.timer.Reset(r.timeout)
	}

	return n, nil
}

// validateDatasetURL exige HTTPS y un host dentro de allowlist. Se usa
// tanto para la petición inicial como, vía CheckRedirect, para cada
// salto de redirección.
func validateDatasetURL(rawURL string, allowlist []string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL inválida: %w", err)
	}

	if parsed.Scheme != "https" {
		return fmt.Errorf("esquema %q no permitido (solo https)", parsed.Scheme)
	}

	if !hostAllowed(parsed.Hostname(), allowlist) {
		return fmt.Errorf("host %q no está en la lista de hosts permitidos", parsed.Hostname())
	}

	return nil
}

// hostAllowed comprueba si host coincide exactamente con una entrada de
// allowlist, o con un patrón de comodín "*.dominio".
func hostAllowed(host string, allowlist []string) bool {
	host = strings.ToLower(host)

	for _, pattern := range allowlist {
		pattern = strings.ToLower(pattern)

		if strings.HasPrefix(pattern, "*.") {
			suffix := pattern[1:]

			if strings.HasSuffix(host, suffix) && len(host) > len(suffix) {
				return true
			}

			continue
		}

		if host == pattern {
			return true
		}
	}

	return false
}

// streamToFile escribe body en spec.Path + ".part" calculando su
// SHA-256 al vuelo, verifica tamaño y hash, y solo entonces lo mueve
// (rename atómico) a spec.Path. Ante cualquier error, elimina el
// archivo parcial: nunca deja un dataset corrupto en su ruta final.
func streamToFile(body io.Reader, spec DatasetSpec, out io.Writer) error {
	dir := filepath.Dir(spec.Path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("%s: no se pudo crear el directorio %s: %w", spec.Name, dir, err)
	}

	partPath := spec.Path + ".part"

	file, err := os.Create(partPath)
	if err != nil {
		return fmt.Errorf("%s: no se pudo crear %s: %w", spec.Name, partPath, err)
	}

	abort := func(causeErr error) error {
		file.Close()
		os.Remove(partPath)
		return causeErr
	}

	hasher := sha256.New()
	progress := newDatasetProgressWriter(spec.Name, spec.Size, out)
	multi := io.MultiWriter(file, hasher, progress)

	// io.LimitReader con Size+1 permite detectar un cuerpo más grande
	// de lo esperado (se leerá un byte de más) sin descargar el
	// archivo completo indefinidamente si el servidor miente.
	limited := io.LimitReader(body, spec.Size+1)

	written, err := io.Copy(multi, limited)
	if err != nil {
		return abort(fmt.Errorf("%s: error al descargar: %w", spec.Name, err))
	}

	if written != spec.Size {
		return abort(fmt.Errorf(
			"%s: se descargaron %d bytes, se esperaban %d (archivo truncado o más grande de lo anunciado)",
			spec.Name, written, spec.Size,
		))
	}

	computedHash := hex.EncodeToString(hasher.Sum(nil))

	if !strings.EqualFold(computedHash, spec.SHA256) {
		return abort(fmt.Errorf(
			"%s: SHA-256 %s no coincide con el esperado %s",
			spec.Name, computedHash, spec.SHA256,
		))
	}

	if err := file.Sync(); err != nil {
		return abort(fmt.Errorf("%s: no se pudo sincronizar el archivo a disco: %w", spec.Name, err))
	}

	if err := file.Close(); err != nil {
		os.Remove(partPath)
		return fmt.Errorf("%s: no se pudo cerrar el archivo descargado: %w", spec.Name, err)
	}

	if err := os.Rename(partPath, spec.Path); err != nil {
		os.Remove(partPath)
		return fmt.Errorf("%s: no se pudo mover el archivo descargado a su destino final: %w", spec.Name, err)
	}

	fmt.Fprintf(out, "%s: descarga completa y verificada en %s\n", spec.Name, spec.Path)

	return nil
}

// datasetProgressWriter imprime el progreso de la descarga cada ~5%,
// sin depender de temporizadores para mantener el comportamiento
// determinista en pruebas.
type datasetProgressWriter struct {
	name       string
	total      int64
	written    int64
	lastBucket int
	out        io.Writer
}

func newDatasetProgressWriter(name string, total int64, out io.Writer) *datasetProgressWriter {
	return &datasetProgressWriter{name: name, total: total, out: out}
}

func (p *datasetProgressWriter) Write(chunk []byte) (int, error) {
	n := len(chunk)
	p.written += int64(n)

	if p.total <= 0 {
		return n, nil
	}

	percent := int(p.written * 100 / p.total)
	bucket := percent / 5

	if bucket > p.lastBucket {
		p.lastBucket = bucket
		fmt.Fprintf(p.out, "%s: %d%%\n", p.name, bucket*5)
	}

	return n, nil
}
