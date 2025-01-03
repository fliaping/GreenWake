package dialog

import (
	"strconv"

	"awake/pkg/i18n"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func ShowTimeInputDialog() (int, error) {
	var result int
	var err error
	done := make(chan struct{})

	// 创建一个新的窗口
	w := fyne.CurrentApp().NewWindow(i18n.T("dialog.time.title"))
	w.Resize(fyne.NewSize(300, 150))

	// 创建输入框
	entry := widget.NewEntry()
	entry.SetPlaceHolder("30")

	// 创建表单
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: i18n.T("dialog.time.message"), Widget: entry},
		},
		OnSubmit: func() {
			val, e := strconv.Atoi(entry.Text)
			if e != nil {
				err = e
			} else {
				result = val
			}
			w.Close()
			close(done)
		},
		OnCancel: func() {
			w.Close()
			close(done)
		},
	}

	// 设置窗口内容
	w.SetContent(container.NewVBox(form))

	// 显示窗口
	w.Show()

	// 等待对话框关闭
	<-done

	return result, err
}
