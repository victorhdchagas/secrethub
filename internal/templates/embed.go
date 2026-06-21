package templates

import "embed"

//go:embed login.html dashboard.html styles.css
var FS embed.FS
