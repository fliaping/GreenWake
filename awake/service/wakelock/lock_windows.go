//go:build windows
// +build windows

package wakelock

import (
	"awake/pkg/logger"

	"golang.org/x/sys/windows"
)

const (
	ES_CONTINUOUS       = 0x80000000
	ES_SYSTEM_REQUIRED  = 0x00000001
	ES_DISPLAY_REQUIRED = 0x00000002
)

type windowsLock struct {
	enabled bool
}

func NewLock() Lock {
	return &windowsLock{}
}

func (l *windowsLock) Acquire() {
	if l.enabled {
		return
	}

	kernel32 := windows.NewLazyDLL("kernel32.dll")
	setThreadExecState := kernel32.NewProc("SetThreadExecutionState")

	ret, _, err := setThreadExecState.Call(
		uintptr(ES_CONTINUOUS | ES_SYSTEM_REQUIRED | ES_DISPLAY_REQUIRED))
	if ret == 0 {
		logger.Error("设置线程执行状态失败: %v", err)
		return
	}

	l.enabled = true
	logger.Info("获取唤醒锁成功")
}

func (l *windowsLock) Release() {
	if !l.enabled {
		return
	}

	kernel32 := windows.NewLazyDLL("kernel32.dll")
	setThreadExecState := kernel32.NewProc("SetThreadExecutionState")

	ret, _, err := setThreadExecState.Call(uintptr(ES_CONTINUOUS))
	if ret == 0 {
		logger.Error("设置线程执行状态失败: %v", err)
		return
	}

	l.enabled = false
	logger.Info("释放唤醒锁成功")
}

func (l *windowsLock) ForceSleep() error {
	// 先释放唤醒锁
	l.Release()

	// 加载 PowrProf.dll
	powrprof := windows.NewLazyDLL("PowrProf.dll")
	setPowerState := powrprof.NewProc("SetSuspendState")

	// 调用 SetSuspendState
	// 参数：
	// - Hibernate: FALSE (使用睡眠而不是休眠)
	// - ForceCritical: FALSE (不强制进入休眠状态)
	// - DisableWakeEvent: FALSE (允许系统被唤醒)
	ret, _, err := setPowerState.Call(0, 0, 0)
	if ret == 0 {
		return err
	}
	return nil
}
