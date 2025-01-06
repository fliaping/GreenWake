//go:build darwin
// +build darwin

package installer

import (
	"fmt"
	"os"
	"path/filepath"
)

type macInstaller struct{}

func (i *macInstaller) Install() error {
	// 获取当前可执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	// 创建启动项
	launchAgentPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.awake.plist")

	// 创建 plist 文件内容
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.awake</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>ProcessType</key>
    <string>Background</string>
</dict>
</plist>`, execPath)

	// 写入 plist 文件
	if err := os.MkdirAll(filepath.Dir(launchAgentPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(launchAgentPath, []byte(plistContent), 0644)
}

func (i *macInstaller) Uninstall() error {
	// 删除启动项
	launchAgentPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.awake.plist")
	return os.Remove(launchAgentPath)
}

func (i *macInstaller) IsInstalled() bool {
	launchAgentPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.awake.plist")
	_, err := os.Stat(launchAgentPath)
	return !os.IsNotExist(err)
}
