// Package apidocs embeds the OpenAPI contract so the binary can serve it under
// /docs in development. A go:embed directive cannot reach a path above the
// module root, so the canonical spec at
// docs/001-capacity-exchange-marketplace/contracts/openapi.yaml cannot be
// embedded directly. Instead a copy lives here, synced from the canonical file
// by apidocs-sync.sh (run locally) and by CI before both go test and the image
// build. The copy is committed so the embed compiles and go test runs on a
// fresh clone; a test asserts byte-for-byte identity with the source so a
// missed sync fails CI instead of shipping a stale spec silently.
//
// The single-file embed (no all: prefix) pulls in only openapi.yaml, so the
// sync script placed outside this directory never lands in the binary.
package apidocs

import _ "embed"

// OpenAPISpec is the raw OpenAPI 3.1 document served at /docs/openapi.yaml.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
