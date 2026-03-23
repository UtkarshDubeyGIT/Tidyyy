package assets

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed app-icon.png
var appIconPNG []byte

// AppIcon is the bundled application icon resource.
var AppIcon fyne.Resource = &fyne.StaticResource{
	StaticName:    "app-icon.png",
	StaticContent: appIconPNG,
}
