// Package templates provides the embedded template filesystem for the renderer.
package templates

import "embed"

// FS contains all .gohtml template files (layouts, partials, pages, admin).
//
//go:embed layouts/*.gohtml partials/*.gohtml pages/*.gohtml admin/*.gohtml
var FS embed.FS
