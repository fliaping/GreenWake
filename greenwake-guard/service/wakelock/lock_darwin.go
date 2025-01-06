//go:build darwin
// +build darwin

package wakelock

import (
	"fmt"
	"os/exec"

	"greenwake-guard/pkg/logger"
)

type darwinLock struct {
	assertionID uint32
}

func NewLock() Lock {
	return &darwinLock{}
}

func (l *darwinLock) Acquire() {
	if l.assertionID != 0 {
		return
	}

	// -i: 防止系统空闲睡眠
	// -d: 防止显示器睡眠
	// -s: 防止系统自动睡眠
	cmd := exec.Command("caffeinate", "-i", "-d", "-s")
	if err := cmd.Start(); err != nil {
		logger.Error("启动 caffeinate 失败: %v", err)
		return
	}

	l.assertionID = uint32(cmd.Process.Pid)
	logger.Info("获取唤醒锁成功，PID: %d", l.assertionID)
}

func (l *darwinLock) Release() {
	if l.assertionID == 0 {
		return
	}

	// 使用进程 ID 精确终止我们的 caffeinate 进程
	cmd := exec.Command("kill", fmt.Sprintf("%d", l.assertionID))
	if err := cmd.Run(); err != nil {
		logger.Error("停止 caffeinate 失败: %v", err)
		return
	}

	l.assertionID = 0
	logger.Info("释放唤醒锁成功")
}

func (l *darwinLock) ForceSleep() error {
	// 先释放唤醒锁
	l.Release()

	// 执行 pmset 命令强制系统睡眠
	cmd := exec.Command("pmset", "sleepnow")
	return cmd.Run()
}
