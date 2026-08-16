package server

import "embed"

//go:embed all:web
var webContent embed.FS
