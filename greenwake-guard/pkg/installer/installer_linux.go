package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type linuxInstaller struct{}

func (i *linuxInstaller) Install() error {
	// 获取当前可执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	// 创建桌面启动项
	autoStartDir := filepath.Join(os.Getenv("HOME"), ".config", "autostart")
	if err := os.MkdirAll(autoStartDir, 0755); err != nil {
		return err
	}

	desktopContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Awake
Comment=System Wake Service
Exec=%s
Icon=awake
Terminal=false
Categories=Utility;
StartupNotify=false`, execPath)

	desktopFile := filepath.Join(autoStartDir, "awake.desktop")
	if err := os.WriteFile(desktopFile, []byte(desktopContent), 0644); err != nil {
		return err
	}

	// 更新桌面数据库
	if err := exec.Command("update-desktop-database").Run(); err != nil {
		return err
	}

	return nil
}

func (i *linuxInstaller) Uninstall() error {
	// 删除桌面启动项
	desktopFile := filepath.Join(os.Getenv("HOME"), ".config", "autostart", "awake.desktop")
	if err := os.Remove(desktopFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	// 更新桌面数据库
	return exec.Command("update-desktop-database").Run()
}

func (i *linuxInstaller) IsInstalled() bool {
	desktopFile := filepath.Join(os.Getenv("HOME"), ".config", "autostart", "awake.desktop")
	_, err := os.Stat(desktopFile)
	return err == nil
}
