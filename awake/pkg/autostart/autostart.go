package autostart

import (
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
}

func NewManager(appName string) (*Manager, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	return &Manager{
		appName:  appName,
		execPath: execPath,
	}, nil
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

func (m *Manager) Disable() error {
	switch runtime.GOOS {
	case "windows":
		return m.disableWindows()
	case "darwin":
		return m.disableMac()
	default:
		return m.disableLinux()
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
	shortcutPath := filepath.Join(startupDir, m.appName+".lnk")

	script := fmt.Sprintf(`
		$WS = New-Object -ComObject WScript.Shell
		$Shortcut = $WS.CreateShortcut("%s")
		$Shortcut.TargetPath = "%s"
		$Shortcut.Save()
	`, shortcutPath, m.execPath)

	cmd := exec.Command("powershell", "-Command", script)
	return cmd.Run()
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

	return os.WriteFile(plistPath, []byte(plistContent), 0644)
}

// Linux 实现
func (m *Manager) enableLinux() error {
	autostartDir := filepath.Join(os.Getenv("HOME"), ".config", "autostart")
	if err := os.MkdirAll(autostartDir, 0755); err != nil {
		return err
	}

	desktopFile := filepath.Join(autostartDir, m.appName+".desktop")
	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s
Terminal=false
Hidden=false`, m.appName, m.execPath)

	return os.WriteFile(desktopFile, []byte(content), 0644)
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
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return exec.Command("launchctl", "unload", plistPath).Run()
}

func (m *Manager) isEnabledMac() bool {
	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", fmt.Sprintf("com.%s.plist", m.appName))
	_, err := os.Stat(plistPath)
	return err == nil
}

// Linux 实现
func (m *Manager) disableLinux() error {
	desktopFile := filepath.Join(os.Getenv("HOME"), ".config", "autostart", m.appName+".desktop")
	return os.Remove(desktopFile)
}

func (m *Manager) isEnabledLinux() bool {
	desktopFile := filepath.Join(os.Getenv("HOME"), ".config", "autostart", m.appName+".desktop")
	_, err := os.Stat(desktopFile)
	return err == nil
}
