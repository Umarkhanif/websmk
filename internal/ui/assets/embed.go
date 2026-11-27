package assets

import "embed"

// Files embeds static UI assets (CSS, JS, vendor HTMX/Alpine, SVG favicon, and
// login photography) directly into the Go binary so deployments need no external
// static folder.
//
//go:embed app.css landing.css app.js favicon.svg manifest.webmanifest icon-*.png apple-touch-icon.png vendor/* images/*.jpg images/*.png fonts/*.woff2
var Files embed.FS
