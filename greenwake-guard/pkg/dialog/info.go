package dialog

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// ShowInfo 显示信息对话框
func ShowInfo(title, message string, parent fyne.Window) {
	d := dialog.NewInformation(title, message, parent)
	d.Resize(fyne.NewSize(400, 300))
	d.Show()
}
