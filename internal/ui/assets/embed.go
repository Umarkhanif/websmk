package assets

import "embed"

// Files embeds static UI assets (CSS, JS, vendor HTMX/Alpine, and SVG favicon)
// directly into the Go binary so deployments need no external static folder.
//
//go:embed app.css app.js favicon.svg vendor/*
var Files embed.FS
