// Package apispec embeds the OpenAPI documents so the server can serve them
// (Swagger UI reads them from /openapi/...).
package apispec

import "embed"

//go:embed common/openapi.yaml auth/openapi.yaml user/openapi.yaml
var FS embed.FS
