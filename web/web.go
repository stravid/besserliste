package web

import (
  "embed"
)

//go:embed "screens"
var Templates embed.FS

//go:embed "static"
var Static embed.FS
