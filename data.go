package defense2

import "embed"

//go:embed config/levels/*.json
var DataFS embed.FS
