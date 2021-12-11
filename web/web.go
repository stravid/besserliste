package web

import (
  "embed"
)

//go:embed "screens"
//go:embed "layouts"
var Templates embed.FS

//go:embed "static"
var Static embed.FS
