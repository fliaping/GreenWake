package tray

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/icon.png
var iconBytes []byte

func getIcon() fyne.Resource {
	return fyne.NewStaticResource("icon.png", iconBytes)
}
