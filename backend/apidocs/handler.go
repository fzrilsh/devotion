package apidocs

import (
	"net/http"

	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// swaggerPage is the Swagger UI shell served at /docs. The assets load from
// jsdelivr pinned to swagger-ui-dist@5.17.14, each with a Subresource Integrity
// hash so a compromised CDN cannot inject code: the browser refuses any asset
// whose bytes do not match the sha384. crossorigin="anonymous" is required for
// SRI to be enforced on cross-origin fetches. The page points Swagger UI at
// /docs/openapi.yaml, the raw contract served from the embedded copy, so the UI
// and the spec always come from the same binary.
//
// The page carries no credentials, tokens, or environment values: it is a
// static shell plus a spec URL (FR: none; constitutional docs obligation).
const swaggerPage = `<!doctype html>
<html lang="id">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Devotion, kontrak API</title>
  <link rel="stylesheet"
    href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui.css"
    integrity="sha384-wxLW6kwyHktdDGr6Pv1zgm/VGJh99lfUbzSn6HNHBENZlCN7W602k9VkGdxuFvPn"
    crossorigin="anonymous">
</head>
<body>
  <div id="swagger-ui"></div>
  <script
    src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"
    integrity="sha384-wmyclcVGX/WhUkdkATwhaK1X1JtiNrr2EoYJ+diV3vj4v6OC5yCeSu+yW13SYJep"
    crossorigin="anonymous"></script>
  <script
    src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui-standalone-preset.js"
    integrity="sha384-2YH8WDRaj7V2OqU/trsmzSagmk/E2SutiCsGkdgoQwC9pNUJV1u/141DHB6jgs8t"
    crossorigin="anonymous"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "/docs/openapi.yaml",
      dom_id: "#swagger-ui",
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
      layout: "StandaloneLayout"
    });
  </script>
</body>
</html>
`

// Register wires the Swagger UI at /docs and the raw contract at
// /docs/openapi.yaml. Both sit outside /api/ so they are exempt from the
// uncovered-route check; Public states the no-auth decision explicitly. The
// caller registers these only in development (see serve), so in production the
// routes are absent and fall to the existing SPA/404 behavior rather than being
// registered and rejected: a rejected route still leaks that the endpoint
// exists.
func Register(r *httpx.Router) {
	r.Public("GET /docs", serveUI)
	r.Public("GET /docs/openapi.yaml", serveSpec)
}

// serveUI returns the Swagger UI shell. It is an exact-path handler: the Go 1.22
// pattern "GET /docs" matches only /docs, not /docs/openapi.yaml, which has its
// own more specific pattern.
func serveUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(swaggerPage))
}

// serveSpec returns the embedded OpenAPI contract. The bytes are the committed
// copy synced from the canonical file, asserted byte-identical to the source by
// a test, so what the UI renders is the same contract CI validates.
func serveSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(OpenAPISpec)
}
