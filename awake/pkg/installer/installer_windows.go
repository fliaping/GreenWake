//go:build windows
// +build windows

package installer

import (
	"fmt"
	"os"

	"awake/pkg/logger"

	"golang.org/x/sys/windows/registry"
)

type windowsInstaller struct{}

func (i *windowsInstaller) Install() error {
	// 获取当前可执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		logger.Error("获取可执行文件路径失败: %v", err)
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	// 添加注册表启动项
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.ALL_ACCESS)
	if err != nil {
		logger.Error("打开注册表键失败: %v", err)
		return fmt.Errorf("打开注册表键失败: %v", err)
	}
	defer k.Close()

	if err := k.SetStringValue("Awake", execPath); err != nil {
		logger.Error("设置注册表值失败: %v", err)
		return fmt.Errorf("设置注册表值失败: %v", err)
	}

	logger.Info("Windows 自启动安装成功")
	return nil
}

func (i *windowsInstaller) Uninstall() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.ALL_ACCESS)
	if err != nil {
		logger.Error("打开注册表键失败: %v", err)
		return fmt.Errorf("打开注册表键失败: %v", err)
	}
	defer k.Close()

	if err := k.DeleteValue("Awake"); err != nil && err != registry.ErrNotExist {
		logger.Error("删除注册表值失败: %v", err)
		return fmt.Errorf("删除注册表值失败: %v", err)
	}

	logger.Info("Windows 自启动卸载成功")
	return nil
}

func (i *windowsInstaller) IsInstalled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	_, _, err = k.GetStringValue("Awake")
	return err == nil
}
