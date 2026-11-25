//go:build ui

package main

import (
	"embed"

	"mgit/pkg/api"
)

//go:embed web/dist
var uiAssets embed.FS

func init() {
	api.UIAssets = uiAssets
}
