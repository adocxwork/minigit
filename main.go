package main

import (
	"embed"
	"mgit/cmd"
	"mgit/pkg/api"
)

//go:embed web/dist
var uiAssets embed.FS

func main() {
	api.UIAssets = uiAssets
	cmd.Execute()
}
