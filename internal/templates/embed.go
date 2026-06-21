package templates

import "embed"

//go:embed login.html setup.html dashboard.html dashboard.js alpine.min.js styles.css
var FS embed.FS
