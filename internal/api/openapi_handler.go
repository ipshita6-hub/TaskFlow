package api

import (
	"embed"
	"net/http"
)

//go:embed openapi.yaml
var openAPISpec embed.FS

// OpenAPIHandler handles GET /api/docs, serving the embedded OpenAPI specification.
func OpenAPIHandler(w http.ResponseWriter, r *http.Request) {
	content, err := openAPISpec.ReadFile("openapi.yaml")
	if err != nil {
		Write500(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}
