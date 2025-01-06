//go:build linux
// +build linux

package wakelock

import (
	"fmt"
	"os"
	"os/exec"

	"greenwake-guard/pkg/logger"
)

type linuxLock struct {
	inhibitCookie uint32
}

func NewLock() Lock {
	return &linuxLock{}
}

func (l *linuxLock) Acquire() {
	if l.inhibitCookie != 0 {
		return
	}

	// 尝试使用 systemd-inhibit
	cmd := exec.Command("systemd-inhibit", "--what=sleep:idle", "--who=awake",
		"--why=Keep system awake", "--mode=block", "sleep", "infinity")
	if err := cmd.Start(); err == nil {
		l.inhibitCookie = uint32(cmd.Process.Pid)
		logger.Info("使用 systemd-inhibit 获取唤醒锁成功，PID: %d", l.inhibitCookie)
		return
	}

	// 如果 systemd-inhibit 失败，尝试使用 xdg-screensaver
	cmd = exec.Command("xdg-screensaver", "suspend", fmt.Sprintf("%d", os.Getpid()))
	if err := cmd.Run(); err == nil {
		l.inhibitCookie = uint32(os.Getpid())
		logger.Info("使用 xdg-screensaver 获取唤醒锁成功")
		return
	}

	logger.Error("获取唤醒锁失败：系统不支持 systemd-inhibit 或 xdg-screensaver")
}

func (l *linuxLock) Release() {
	if l.inhibitCookie == 0 {
		return
	}

	// 尝试终止 systemd-inhibit 进程
	if err := exec.Command("kill", fmt.Sprintf("%d", l.inhibitCookie)).Run(); err != nil {
		// 如果失败，可能是使用的 xdg-screensaver
		if err := exec.Command("xdg-screensaver", "resume", fmt.Sprintf("%d", l.inhibitCookie)).Run(); err != nil {
			logger.Error("释放唤醒锁失败: %v", err)
			return
		}
	}

	l.inhibitCookie = 0
	logger.Info("释放唤醒锁成功")
}

func (l *linuxLock) ForceSleep() error {
	// 先释放唤醒锁
	l.Release()

	// 尝试使用 systemctl 命令
	if err := exec.Command("systemctl", "suspend").Run(); err == nil {
		return nil
	}

	// 如果 systemctl 失败，尝试使用 pm-suspend
	if err := exec.Command("pm-suspend").Run(); err == nil {
		return nil
	}

	// 最后尝试使用 dbus-send
	cmd := exec.Command("dbus-send", "--system", "--print-reply",
		"--dest=org.freedesktop.login1",
		"/org/freedesktop/login1",
		"org.freedesktop.login1.Manager.Suspend",
		"boolean:true")
	return cmd.Run()
}
