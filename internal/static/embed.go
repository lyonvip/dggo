package static

import "embed"

//go:embed *
var StaticSource embed.FS
