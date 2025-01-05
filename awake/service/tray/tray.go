package tray

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"awake/config"
	"awake/pkg/autostart"
	"awake/pkg/dialog"
	"awake/pkg/i18n"
	"awake/pkg/logger"
	"awake/service/wakelock"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"gopkg.in/yaml.v3"
)

type TrayService struct {
	wakeLockService *wakelock.Service
	currentMode     string
	autoStartMgr    *autostart.Manager
	config          *config.Config
	configPath      string
	app             fyne.App
	window          fyne.Window
	desk            desktop.App
	menu            *fyne.Menu
	modeItem        *fyne.MenuItem
	remainingTime   string
}

func NewTrayService(wakeLockService *wakelock.Service) *TrayService {
	autoStartMgr, err := autostart.NewManager("awake")
	if err != nil {
		logger.Error("创建自启动管理器失败: %v", err)
	}

	s := &TrayService{
		wakeLockService: wakeLockService,
		currentMode:     "packet",
		autoStartMgr:    autoStartMgr,
	}

	// 设置更新回调函数，用于更新UI显示
	wakeLockService.SetUpdateCallback(func() {
		if s.desk != nil {
			s.desk.SetSystemTrayMenu(s.createMenu())
		}
	})

	return s
}

func (s *TrayService) createMenu() *fyne.Menu {
	// 更新剩余时间
	s.remainingTime = s.wakeLockService.FormatRemainingTime()

	// 创建唤醒时间子菜单
	timeMenu := fyne.NewMenu(i18n.T("menu.time"),
		&fyne.MenuItem{
			Label:   i18n.T("menu.time.30min"),
			Checked: s.wakeLockService.GetStrategy() == wakelock.StrategyTimed && s.wakeLockService.GetDuration() == 30*time.Minute,
			Action: func() {
				s.wakeLockService.SetStrategy(wakelock.StrategyTimed, 30*time.Minute)
			},
		},
		&fyne.MenuItem{
			Label:   i18n.T("menu.time.1hour"),
			Checked: s.wakeLockService.GetStrategy() == wakelock.StrategyTimed && s.wakeLockService.GetDuration() == time.Hour,
			Action: func() {
				s.wakeLockService.SetStrategy(wakelock.StrategyTimed, time.Hour)
			},
		},
		&fyne.MenuItem{
			Label:   i18n.T("menu.time.2hour"),
			Checked: s.wakeLockService.GetStrategy() == wakelock.StrategyTimed && s.wakeLockService.GetDuration() == 2*time.Hour,
			Action: func() {
				s.wakeLockService.SetStrategy(wakelock.StrategyTimed, 2*time.Hour)
			},
		},
		&fyne.MenuItem{
			Label:   i18n.T("menu.time.4hour"),
			Checked: s.wakeLockService.GetStrategy() == wakelock.StrategyTimed && s.wakeLockService.GetDuration() == 4*time.Hour,
			Action: func() {
				s.wakeLockService.SetStrategy(wakelock.StrategyTimed, 4*time.Hour)
			},
		},
		fyne.NewMenuItemSeparator(),
		&fyne.MenuItem{
			Label: i18n.T("menu.time.custom"),
			Checked: s.wakeLockService.GetStrategy() == wakelock.StrategyTimed &&
				s.wakeLockService.GetDuration() != 15*time.Minute &&
				s.wakeLockService.GetDuration() != 30*time.Minute &&
				s.wakeLockService.GetDuration() != time.Hour &&
				s.wakeLockService.GetDuration() != 2*time.Hour &&
				s.wakeLockService.GetDuration() != 4*time.Hour,
			Action: func() {
				minutes, err := dialog.ShowTimeInputDialog()
				if err != nil {
					logger.Error("显示时间选择对话框失败: %v", err)
					return
				}
				if minutes > 0 {
					duration := time.Duration(minutes) * time.Minute
					s.wakeLockService.SetStrategy(wakelock.StrategyTimed, duration)
				}
			},
		},
	)

	// 创建睡眠模式子菜单
	sleepMenu := fyne.NewMenu(i18n.T("menu.sleep"),
		&fyne.MenuItem{
			Label:   i18n.T("menu.sleep.system"),
			Checked: s.wakeLockService.GetSleepMode() == wakelock.SleepModeSystem,
			Action: func() {
				s.wakeLockService.SetSleepMode(wakelock.SleepModeSystem)
			},
		},
		&fyne.MenuItem{
			Label:   i18n.T("menu.sleep.program"),
			Checked: s.wakeLockService.GetSleepMode() == wakelock.SleepModeProgram,
			Action: func() {
				s.wakeLockService.SetSleepMode(wakelock.SleepModeProgram)
			},
		},
	)

	// 获取当前策略和剩余时间
	timedLabel := i18n.T("menu.timed")
	if s.wakeLockService.GetStrategy() == wakelock.StrategyTimed {
		timedLabel += s.remainingTime
	}

	// 创建唤醒模式状态信息
	var wakeStatusLabel string
	wakeStatusLabel = i18n.T("menu.status.wake") + ": "

	// 记录当前策略
	logger.Debug("当前策略: %v", s.wakeLockService.GetStrategy())

	switch s.wakeLockService.GetStrategy() {
	case wakelock.StrategyWolWake:
		wakeStatusLabel += i18n.T("menu.wol_wake")
	case wakelock.StrategyPermanent:
		wakeStatusLabel += i18n.T("menu.permanent")
	case wakelock.StrategyTimed:
		wakeStatusLabel += i18n.T("menu.timed")
		remaining := s.remainingTime
		logger.Debug("计时唤醒剩余时间: %v", remaining)
		if remaining != "" {
			wakeStatusLabel += remaining
		}
		logger.Debug("最终的唤醒状态标签: %v", wakeStatusLabel)
	}

	// 创建睡眠模式状态信息
	var sleepStatusLabel string
	sleepStatusLabel = i18n.T("menu.status.sleep") + ": "
	if s.wakeLockService.GetSleepMode() == wakelock.SleepModeSystem {
		sleepStatusLabel += i18n.T("menu.sleep.system")
	} else {
		sleepStatusLabel += i18n.T("menu.sleep.program")
	}

	// 创建主菜单
	menuItems := []*fyne.MenuItem{
		&fyne.MenuItem{
			Label:   wakeStatusLabel,
			Checked: false,
			Action:  nil,
		},
		&fyne.MenuItem{
			Label:   sleepStatusLabel,
			Checked: false,
			Action:  nil,
		},
		fyne.NewMenuItemSeparator(),
		&fyne.MenuItem{
			Label:   i18n.T("menu.wol_wake"),
			Checked: s.wakeLockService.GetStrategy() == wakelock.StrategyWolWake,
			Action: func() {
				s.wakeLockService.SetStrategy(wakelock.StrategyWolWake, 0)
			},
		},
		&fyne.MenuItem{
			Label:   i18n.T("menu.permanent"),
			Checked: s.wakeLockService.GetStrategy() == wakelock.StrategyPermanent,
			Action: func() {
				s.wakeLockService.SetStrategy(wakelock.StrategyPermanent, 0)
			},
		},
		&fyne.MenuItem{
			Label:     timedLabel,
			Checked:   s.wakeLockService.GetStrategy() == wakelock.StrategyTimed,
			ChildMenu: timeMenu,
		},
		fyne.NewMenuItemSeparator(),
		&fyne.MenuItem{
			Label:     i18n.T("menu.sleep"),
			ChildMenu: sleepMenu,
		},
		fyne.NewMenuItemSeparator(),
		&fyne.MenuItem{
			Label: i18n.T("menu.show_prevent_sleep"),
			Action: func() {
				dialog.ShowPreventSleepProcesses(s.window, s.wakeLockService)
			},
		},
		fyne.NewMenuItemSeparator(),
		&fyne.MenuItem{
			Label: i18n.T("menu.quit"),
			Action: func() {
				s.app.Quit()
			},
		},
	}

	menu := fyne.NewMenu(i18n.T("app.name"), menuItems...)
	s.menu = menu // 保存菜单引用
	return menu
}

func (s *TrayService) Start() {
	// 初始化 i18n
	langDir := filepath.Join("assets", "lang")
	if _, err := os.Stat(langDir); os.IsNotExist(err) {
		// 如果相对路径不存在，尝试使用可执行文件路径
		execPath, err := os.Executable()
		if err != nil {
			logger.Error("获取可执行文件路径失败: %v", err)
			return
		}
		baseDir := filepath.Dir(execPath)
		langDir = filepath.Join(baseDir, "assets", "lang")
	}
	if err := i18n.Init(langDir); err != nil {
		logger.Error("初始化 i18n 失败: %v", err)
		return
	}

	s.app = app.NewWithID("com.fliaping.awake")
	s.window = s.app.NewWindow(i18n.T("app.name"))
	s.window.SetIcon(getIcon())
	s.window.Resize(fyne.NewSize(300, 200))

	// 设置系统托盘菜单
	var ok bool
	if s.desk, ok = s.app.(desktop.App); ok {
		// 设置托盘图标
		s.desk.SetSystemTrayIcon(getIcon())
		// 设置托盘菜单
		s.desk.SetSystemTrayMenu(s.createMenu())
	}

	// 设置窗口关闭行为
	s.window.SetCloseIntercept(func() {
		s.window.Hide()
	})

	// 运行应用
	s.window.Hide() // 初始时隐藏窗口
	s.app.Run()
}

func (s *TrayService) updateModeStatus() {
	strategy := s.wakeLockService.GetStrategy()
	sleepMode := s.wakeLockService.GetSleepMode()
	duration := s.wakeLockService.GetDuration()

	// 更新菜单项选中状态
	for _, item := range s.menu.Items {
		switch item.Label {
		case i18n.T("menu.wol_wake"):
			item.Checked = strategy == wakelock.StrategyWolWake
		case i18n.T("menu.permanent"):
			item.Checked = strategy == wakelock.StrategyPermanent
		case i18n.T("menu.timed"):
			item.Checked = strategy == wakelock.StrategyTimed
			// 更新计时唤醒子菜单的选中状态
			if item.ChildMenu != nil {
				for _, child := range item.ChildMenu.Items {
					if child.Label == "" { // 分隔符
						continue
					}
					switch child.Label {
					case i18n.T("menu.time.30min"):
						child.Checked = strategy == wakelock.StrategyTimed && duration == 30*time.Minute
					case i18n.T("menu.time.1hour"):
						child.Checked = strategy == wakelock.StrategyTimed && duration == time.Hour
					case i18n.T("menu.time.2hour"):
						child.Checked = strategy == wakelock.StrategyTimed && duration == 2*time.Hour
					case i18n.T("menu.time.4hour"):
						child.Checked = strategy == wakelock.StrategyTimed && duration == 4*time.Hour
					case i18n.T("menu.time.custom"):
						isCustomDuration := strategy == wakelock.StrategyTimed &&
							duration != 30*time.Minute &&
							duration != time.Hour &&
							duration != 2*time.Hour &&
							duration != 4*time.Hour
						child.Checked = isCustomDuration
						if isCustomDuration {
							child.Label = fmt.Sprintf("%s (%s)", i18n.T("menu.time.custom"), duration.String())
						} else {
							child.Label = i18n.T("menu.time.custom")
						}
					}
				}
			}
		case i18n.T("menu.sleep"):
			// 更新睡眠模式子菜单的选中状态
			if item.ChildMenu != nil {
				for _, child := range item.ChildMenu.Items {
					switch child.Label {
					case i18n.T("menu.sleep.system"):
						child.Checked = sleepMode == wakelock.SleepModeSystem
					case i18n.T("menu.sleep.program"):
						child.Checked = sleepMode == wakelock.SleepModeProgram
					}
				}
			}
		case i18n.T("menu.autostart"):
			item.Checked = s.autoStartMgr.IsEnabled()
		}
	}

	// 更新托盘菜单
	if s.desk != nil {
		s.desk.SetSystemTrayMenu(s.menu)
	}
}

func (s *TrayService) Stop() {
	if s.app != nil {
		s.app.Quit()
	}
}

func (s *TrayService) SetConfig(cfg *config.Config, configPath string) {
	logger.Debug("[SetConfig] 开始设置配置: path=%s, config=%+v", configPath, cfg)
	s.config = cfg
	s.configPath = configPath

	// 设置默认值
	if cfg.Initial.Strategy == "" {
		cfg.Initial.Strategy = string(wakelock.StrategyWolWake)
	}
	if cfg.Initial.SleepMode == "" {
		cfg.Initial.SleepMode = string(wakelock.SleepModeProgram)
	}

	// 设置初始策略和睡眠模式
	logger.Info("已从配置文件加载策略: %s, 睡眠模式: %s", cfg.Initial.Strategy, cfg.Initial.SleepMode)
	s.wakeLockService.InitializeState(
		wakelock.Strategy(cfg.Initial.Strategy),
		wakelock.SleepMode(cfg.Initial.SleepMode),
		parseDuration(cfg.Initial.Duration),
	)

	// 更新菜单项状态
	s.updateMenuItemStates(
		wakelock.Strategy(cfg.Initial.Strategy),
		wakelock.SleepMode(cfg.Initial.SleepMode),
		parseDuration(cfg.Initial.Duration),
	)
}

func (s *TrayService) SaveConfig() error {
	logger.Debug("[SaveConfig] 开始保存配置")
	if s.config == nil {
		return fmt.Errorf("配置为空")
	}

	// 更新配置
	s.config.Initial.Strategy = string(s.wakeLockService.GetStrategy())
	s.config.Initial.SleepMode = string(s.wakeLockService.GetSleepMode())
	if s.wakeLockService.GetStrategy() == wakelock.StrategyTimed {
		s.config.Initial.Duration = formatDuration(s.wakeLockService.GetDuration())
	}

	// 保存配置
	data, err := yaml.Marshal(s.config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	// 确保配置目录存在
	if err := os.MkdirAll(filepath.Dir(s.configPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(s.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	logger.Debug("[SaveConfig] 配置保存成功")
	return nil
}

// parseDuration 解析持续时间字符串
func parseDuration(durationStr string) time.Duration {
	if durationStr == "" {
		return 30 * time.Minute // 默认30分钟
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		logger.Error("解析持续时间失败: %v，使用默认值30分钟", err)
		return 30 * time.Minute
	}
	return duration
}

// formatDuration 格式化持续时间
func formatDuration(duration time.Duration) string {
	return duration.String()
}

// updateMenuItemStates 更新菜单项的状态
func (s *TrayService) updateMenuItemStates(strategy wakelock.Strategy, sleepMode wakelock.SleepMode, duration time.Duration) {
	if s.menu == nil {
		return
	}

	// 更新模式状态
	s.currentMode = string(strategy)
	if s.modeItem != nil {
		s.modeItem.Checked = true
	}

	// 更新剩余时间显示
	if duration > 0 {
		s.remainingTime = formatDuration(duration)
	}
}
