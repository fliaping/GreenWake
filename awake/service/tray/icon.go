package tray

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/icon.ico
var iconBytes []byte

func getIcon() fyne.Resource {
	return fyne.NewStaticResource("icon.ico", iconBytes)
}
