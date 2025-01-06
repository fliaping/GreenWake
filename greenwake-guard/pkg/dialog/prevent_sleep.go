package dialog

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"greenwake-guard/pkg/i18n"
	"greenwake-guard/pkg/logger"
	"greenwake-guard/pkg/system"
	"greenwake-guard/service/wakelock"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ShowPreventSleepProcesses 显示阻止休眠的进程列表
func ShowPreventSleepProcesses(parent fyne.Window, wakeLockService *wakelock.Service) {
	processes, powerState, kernelAssertions, err := system.GetPreventSleepProcesses()
	if err != nil {
		logger.Error("获取阻止休眠进程失败: %v", err)
		return
	}

	// 创建一个新窗口
	w := fyne.CurrentApp().NewWindow(i18n.T("menu.prevent_sleep_title"))
	w.Resize(fyne.NewSize(900, 600)) // 设置更合理的初始窗口大小

	// 创建系统状态展示
	sysStatus := createSystemStatusDisplay(powerState)

	// 创建进程表格
	procTable := createProcessTable(processes, wakeLockService)

	// 创建内核断言展示
	kernelStatus := createKernelAssertionsDisplay(kernelAssertions)

	// 创建说明文本
	helpText := widget.NewRichTextFromMarkdown(`**休眠阻止类型说明：**
- 系统休眠阻止：阻止系统进入休眠状态
- 用户空闲休眠阻止：阻止系统在用户空闲时休眠
- 显示器休眠阻止：阻止显示器进入睡眠状态

**常见原因说明：**
- 后台活动：系统正在执行后台任务
- 外部设备活动：外部设备正在使用中
- 网络活动：网络连接正在使用中
- 用户活动：检测到用户正在活动

**超时动作说明：**
- TimeoutActionRelease：超时后释放休眠阻止
- TimeoutActionTurnOff：超时后关闭显示器`)

	// 使用选项卡组织内容
	tabs := container.NewAppTabs(
		container.NewTabItem("系统状态", sysStatus),
		container.NewTabItem("进程信息", procTable),
		container.NewTabItem("内核断言", kernelStatus),
	)

	// 创建主布局
	content := container.NewBorder(
		helpText,
		nil,
		nil,
		nil,
		tabs,
	)

	// 创建状态文本
	var statusText string
	switch wakeLockService.GetStrategy() {
	case wakelock.StrategyExternalWake:
		if len(processes) == 0 {
			statusText = i18n.T("menu.no_prevent_sleep")
		} else {
			var processNames []string
			for _, p := range processes {
				processNames = append(processNames, p.Name)
			}
			statusText = i18n.T("menu.prevent_sleep_list") + "\n" + strings.Join(processNames, "\n")
		}
	default:
		statusText = i18n.T("menu.no_prevent_sleep")
	}

	// 添加状态文本到布局
	statusLabel := widget.NewLabel(statusText)
	content = container.NewBorder(
		container.NewVBox(helpText, widget.NewSeparator(), statusLabel),
		nil,
		nil,
		nil,
		tabs,
	)

	w.SetContent(content)
	w.Show()
}

func createSystemStatusDisplay(ps *system.SystemPowerState) fyne.CanvasObject {
	if ps == nil {
		return widget.NewLabelWithStyle(
			"无法获取系统电源状态",
			fyne.TextAlignCenter,
			fyne.TextStyle{},
		)
	}

	descriptions := system.GetPowerStateDescription(ps)
	items := []string{
		fmt.Sprintf("%s: %v", descriptions["PreventSystemSleep"], formatBool(ps.PreventSystemSleep)),
		fmt.Sprintf("%s: %v", descriptions["PreventUserIdle"], formatBool(ps.PreventUserIdle)),
		fmt.Sprintf("%s: %v", descriptions["PreventDisplaySleep"], formatBool(ps.PreventDisplaySleep)),
		fmt.Sprintf("%s: %v", descriptions["BackgroundActivity"], formatBool(ps.BackgroundActivity)),
		fmt.Sprintf("%s: %v", descriptions["ExternalDevice"], formatBool(ps.ExternalDevice)),
		fmt.Sprintf("%s: %v", descriptions["NetworkActivity"], formatBool(ps.NetworkActivity)),
	}

	grid := container.NewGridWithColumns(2)
	for _, item := range items {
		grid.Add(widget.NewLabel(item))
	}

	return container.NewVBox(
		widget.NewLabelWithStyle("系统电源状态概览", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		grid,
	)
}

func createProcessTable(processes []system.PreventSleepProcess, wakeLockService *wakelock.Service) fyne.CanvasObject {
	// 获取当前进程状态
	currentProcess := system.GetCurrentProcessState()

	// 如果没有任何进程且当前程序也没有阻止休眠
	if len(processes) == 0 && currentProcess == nil {
		return widget.NewLabelWithStyle(
			"当前没有进程阻止系统休眠",
			fyne.TextAlignCenter,
			fyne.TextStyle{},
		)
	}

	// 创建进程列表（包括系统进程和本程序）
	displayProcesses := make([]system.PreventSleepProcess, 0, len(processes)+1)

	// 添加本程序的状态（如果正在阻止休眠）
	strategy := wakeLockService.GetStrategy()
	if strategy == wakelock.StrategyPermanent || strategy == wakelock.StrategyTimed ||
		(strategy == wakelock.StrategyExternalWake && wakeLockService.GetRemainingTime() > 0) {
		// 检查是否已经在系统进程列表中
		found := false
		for _, p := range processes {
			if p.PID == os.Getpid() {
				found = true
				break
			}
		}
		// 如果不在系统进程列表中，添加到显示列表
		if !found {
			var details string
			switch strategy {
			case wakelock.StrategyPermanent:
				details = "当前唤醒策略：永久唤醒"
			case wakelock.StrategyTimed:
				details = fmt.Sprintf("当前唤醒策略：计时唤醒%s", wakeLockService.FormatRemainingTime())
			case wakelock.StrategyExternalWake:
				details = fmt.Sprintf("当前唤醒策略：外部唤醒%s", wakeLockService.FormatRemainingTime())
			}

			awakeProcess := system.PreventSleepProcess{
				PID:     os.Getpid(),
				Name:    "当前进程",
				Type:    "PreventUserIdleSystemSleep",
				Reason:  "保持系统唤醒",
				Details: details,
			}
			displayProcesses = append(displayProcesses, awakeProcess)
		}
	}

	// 添加系统进程列表
	displayProcesses = append(displayProcesses, processes...)

	table := widget.NewTable(
		func() (int, int) {
			return len(displayProcesses) + 1, 5 // +1 for header
		},
		func() fyne.CanvasObject {
			return widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{})
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			label.Wrapping = fyne.TextTruncate
			label.TextStyle = fyne.TextStyle{}

			if i.Row == 0 {
				// 表头
				label.TextStyle.Bold = true
				label.Alignment = fyne.TextAlignCenter
				switch i.Col {
				case 0:
					label.SetText("PID")
				case 1:
					label.SetText("进程名")
				case 2:
					label.SetText("持续时间")
				case 3:
					label.SetText("类型")
				case 4:
					label.SetText("详细信息")
				}
			} else {
				// 数据行
				p := displayProcesses[i.Row-1]
				label.Alignment = fyne.TextAlignLeading
				switch i.Col {
				case 0:
					label.SetText(strconv.Itoa(p.PID))
					label.Alignment = fyne.TextAlignCenter
				case 1:
					label.SetText(p.Name)
					if p.Name == "当前进程" || (currentProcess != nil && p.PID == currentProcess.PID) {
						label.TextStyle.Bold = true
					}
				case 2:
					label.SetText(p.Duration)
					label.Alignment = fyne.TextAlignCenter
				case 3:
					typeDesc := system.GetProcessDescription(p)
					label.SetText(typeDesc)
				case 4:
					if p.Name == "当前进程" {
						label.SetText(p.Details)
					} else {
						label.SetText(system.GetProcessDetailInfo(p))
					}
				}
			}
		})

	// 设置列宽比例
	totalWidth := float32(900)               // 基准总宽度
	table.SetColumnWidth(0, totalWidth*0.08) // PID: 8%
	table.SetColumnWidth(1, totalWidth*0.17) // 进程名: 17%
	table.SetColumnWidth(2, totalWidth*0.12) // 持续时间: 12%
	table.SetColumnWidth(3, totalWidth*0.18) // 类型: 18%
	table.SetColumnWidth(4, totalWidth*0.45) // 详细信息: 45%

	// 创建一个可滚动的容器来包装表格
	scroll := container.NewScroll(table)

	// 使用 Border 布局包装滚动容器，这样可以更好地适应窗口大小变化
	return container.NewBorder(nil, nil, nil, nil, scroll)
}

func createKernelAssertionsDisplay(assertions []system.KernelPowerAssertion) fyne.CanvasObject {
	if len(assertions) == 0 {
		return widget.NewLabelWithStyle(
			"当前没有内核级休眠阻止",
			fyne.TextAlignCenter,
			fyne.TextStyle{},
		)
	}

	content := widget.NewTextGrid()
	text := "内核级休眠阻止：\n\n"
	for _, ka := range assertions {
		text += fmt.Sprintf("ID: %d (级别: %d)\n", ka.ID, ka.Level)
		text += fmt.Sprintf("类型: %s\n", ka.Type)
		text += fmt.Sprintf("描述: %s\n", ka.Description)
		text += fmt.Sprintf("所有者: %s\n", ka.Owner)
		if ka.CreateTime != "" {
			text += fmt.Sprintf("创建时间: %s\n", ka.CreateTime)
		}
		if ka.ModTime != "" {
			text += fmt.Sprintf("修改时间: %s\n", ka.ModTime)
		}
		text += "\n"
	}
	content.SetText(text)

	return container.NewScroll(content)
}

func formatBool(b bool) string {
	if b {
		return "是"
	}
	return "否"
}
