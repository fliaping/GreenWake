package autostart

import (
	"awake/pkg/logger"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Manager 自启动管理器
type Manager struct {
	appName   string
	execPath  string
	autoStart bool
	onQuit    func() // 退出回调函数
}

func NewManager(appName string, onQuit func()) (*Manager, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	return &Manager{
		appName:  appName,
		execPath: execPath,
		onQuit:   onQuit,
	}, nil
}

// IsLaunchedByService 检查当前进程是否由服务启动
func (m *Manager) IsLaunchedByService() bool {
	switch runtime.GOOS {
	case "darwin":
		// 检查是否由 launchd 启动
		ppid := os.Getppid()
		out, err := exec.Command("ps", "-p", fmt.Sprintf("%d", ppid), "-o", "comm=").Output()
		if err != nil {
			return false
		}
		return string(out) == "launchd\n"
	case "linux":
		// 检查是否由 systemd 启动
		ppid := os.Getppid()
		out, err := exec.Command("ps", "-p", fmt.Sprintf("%d", ppid), "-o", "comm=").Output()
		if err != nil {
			return false
		}
		return string(out) == "systemd\n"
	default:
		return false
	}
}

// Disable 禁用自启动
func (m *Manager) Disable() error {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = m.disableWindows()
	case "darwin":
		err = m.disableMac()
	default:
		err = m.disableLinux()
	}
	return err
}

func (m *Manager) Enable() error {
	switch runtime.GOOS {
	case "windows":
		return m.enableWindows()
	case "darwin":
		return m.enableMac()
	default:
		return m.enableLinux()
	}
}

func (m *Manager) IsEnabled() bool {
	switch runtime.GOOS {
	case "windows":
		return m.isEnabledWindows()
	case "darwin":
		return m.isEnabledMac()
	default:
		return m.isEnabledLinux()
	}
}

// Windows 实现
func (m *Manager) enableWindows() error {
	startupDir := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	// 确保目录存在
	if err := os.MkdirAll(startupDir, 0755); err != nil {
		return fmt.Errorf("创建启动目录失败: %v", err)
	}

	shortcutPath := filepath.Join(startupDir, m.appName+".lnk")
	workingDir := filepath.Dir(m.execPath)

	script := fmt.Sprintf(`
		$WS = New-Object -ComObject WScript.Shell
		$Shortcut = $WS.CreateShortcut("%s")
		$Shortcut.TargetPath = "%s"
		$Shortcut.WorkingDirectory = "%s"
		$Shortcut.Description = "系统唤醒服务"
		$Shortcut.WindowStyle = 7
		$Shortcut.Save()
	`, shortcutPath, m.execPath, workingDir)

	cmd := exec.Command("powershell", "-Command", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("创建快捷方式失败: %v", err)
	}
	return nil
}

// macOS 实现
func (m *Manager) enableMac() error {
	launchAgentsDir := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
	plistPath := filepath.Join(launchAgentsDir, fmt.Sprintf("com.%s.plist", m.appName))

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>`, m.appName, m.execPath)

	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("写入 plist 文件失败: %v", err)
	}

	// 加载 plist
	cmd := exec.Command("launchctl", "load", plistPath)
	if err := cmd.Run(); err != nil {
		os.Remove(plistPath) // 如果加载失败，清理文件
		return fmt.Errorf("加载 plist 失败: %v", err)
	}

	return nil
}

// Linux 实现
func (m *Manager) enableLinux() error {
	systemdDir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		return fmt.Errorf("创建 systemd 用户目录失败: %v", err)
	}

	serviceFile := filepath.Join(systemdDir, fmt.Sprintf("%s.service", m.appName))
	content := fmt.Sprintf(`[Unit]
Description=系统唤醒服务
After=network.target

[Service]
Type=simple
ExecStart=%s
WorkingDirectory=%s
Restart=always
RestartSec=10

[Install]
WantedBy=default.target`, m.execPath, filepath.Dir(m.execPath))

	if err := os.WriteFile(serviceFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入 service 文件失败: %v", err)
	}

	// 重新加载 systemd 用户配置
	reloadCmd := exec.Command("systemctl", "--user", "daemon-reload")
	if err := reloadCmd.Run(); err != nil {
		return fmt.Errorf("重新加载 systemd 配置失败: %v", err)
	}

	// 启用服务
	enableCmd := exec.Command("systemctl", "--user", "enable", fmt.Sprintf("%s.service", m.appName))
	if err := enableCmd.Run(); err != nil {
		return fmt.Errorf("启用 systemd 服务失败: %v", err)
	}

	return nil
}

// Windows 实现
func (m *Manager) disableWindows() error {
	startupDir := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	shortcutPath := filepath.Join(startupDir, m.appName+".lnk")
	return os.Remove(shortcutPath)
}

func (m *Manager) isEnabledWindows() bool {
	startupDir := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	shortcutPath := filepath.Join(startupDir, m.appName+".lnk")
	_, err := os.Stat(shortcutPath)
	return err == nil
}

// macOS 实现
func (m *Manager) disableMac() error {
	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", fmt.Sprintf("com.%s.plist", m.appName))

	// 先停止服务
	if err := exec.Command("launchctl", "stop", fmt.Sprintf("com.%s", m.appName)).Run(); err != nil {
		logger.Debug("停止服务失败: %v", err)
	}

	// 再卸载服务
	if err := exec.Command("launchctl", "unload", "-w", plistPath).Run(); err != nil {
		logger.Debug("卸载服务失败: %v", err)
	}

	// 最后删除文件
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除 plist 文件失败: %v", err)
	}

	return nil
}

func (m *Manager) isEnabledMac() bool {
	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", fmt.Sprintf("com.%s.plist", m.appName))
	_, err := os.Stat(plistPath)
	return err == nil
}

// Linux 禁用自启动
func (m *Manager) disableLinux() error {
	serviceName := fmt.Sprintf("%s.service", m.appName)

	// 先停止服务
	stopCmd := exec.Command("systemctl", "--user", "stop", serviceName)
	if err := stopCmd.Run(); err != nil {
		logger.Debug("停止服务失败: %v", err)
	}

	// 禁用服务
	disableCmd := exec.Command("systemctl", "--user", "disable", serviceName)
	if err := disableCmd.Run(); err != nil {
		logger.Debug("禁用服务失败: %v", err)
	}

	// 重新加载配置
	reloadCmd := exec.Command("systemctl", "--user", "daemon-reload")
	if err := reloadCmd.Run(); err != nil {
		logger.Debug("重新加载配置失败: %v", err)
	}

	// 删除服务文件
	serviceFile := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", serviceName)
	if err := os.Remove(serviceFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除 service 文件失败: %v", err)
	}

	return nil
}

// Linux 检查自启动状态
func (m *Manager) isEnabledLinux() bool {
	cmd := exec.Command("systemctl", "--user", "is-enabled", fmt.Sprintf("%s.service", m.appName))
	return cmd.Run() == nil
}
