package server

import (
	_ "embed"
	"net/http"
)

//go:embed web/index.html
var indexHTML []byte

//go:embed web/openapi.yaml
var openAPISpec []byte

// HandleIndex serves the user-friendly Web UI.
func HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(indexHTML)
}

// HandleOpenAPISpec serves raw OpenAPI 3.0 specification.
func HandleOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openAPISpec)
}

// HandleDocs serves interactive Scalar API reference documentation.
func HandleDocs(w http.ResponseWriter, _ *http.Request) {
	const scalarHTML = `<!doctype html>
<html>
  <head>
    <title>lumidive API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <link rel="icon" href="data:,">
  </head>
  <body>
    <script id="api-reference" data-url="/openapi.yaml"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(scalarHTML))
}
