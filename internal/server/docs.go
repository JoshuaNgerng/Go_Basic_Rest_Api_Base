package server

import (
	"net/http"

	apispec "authapi/api"
)

// serveSpecs serves the embedded OpenAPI files under /openapi/, keeping the
// folder layout (/openapi/auth/openapi.yaml, /openapi/common/openapi.yaml, ...)
// so the relative $refs between specs resolve in Swagger UI.
func serveSpecs() http.Handler {
	return http.StripPrefix("/openapi/", http.FileServerFS(apispec.FS))
}

const swaggerHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <title>API docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
	window.ui = SwaggerUIBundle({
      urls: [
		{ url: "/openapi/auth/openapi.yaml", name: "Auth" },
		{ url: "/openapi/user/openapi.yaml", name: "User" }
      ],
      "urls.primaryName": "Auth",
      dom_id: "#swagger-ui",
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
      layout: "StandaloneLayout",
      persistAuthorization: true
	});
  </script>
</body>
</html>`

func serveSwaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerHTML))
}
