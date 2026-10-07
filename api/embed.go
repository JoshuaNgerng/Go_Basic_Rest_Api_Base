// Package apispec exposes the OpenAPI document so the server can serve it.
package apispec

import _ "embed"

//go:embed openapi.yaml
var Spec []byte
