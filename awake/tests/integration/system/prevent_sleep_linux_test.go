//go:build linux
// +build linux

package system_test

import (
	"awake/pkg/system"
	"os/exec"
	"testing"
)

// mockCommand 模拟 Linux systemd-inhibit 命令输出
var mockSystemdInhibitOutput = `WHO                          UID  PID  COMM      WHAT     WHY                                                      MODE
user                         1000 1234 firefox   idle     Firefox is playing audio                               delay
root                         0    5678 systemd   sleep    System suspend/hibernate                              block
user                         1000 9012 vlc       idle     Video playback                                        block`

// mockLoginctlOutput 模拟 loginctl 命令输出
var mockLoginctlOutput = `IdleHint=no
IdleSinceHint=0
IdleSinceHintMonotonic=0`

// TestLinuxPowerInhibitors 测试 Linux 平台电源抑制器信息获取
func TestLinuxPowerInhibitors(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Linux power inhibitors test in short mode")
	}

	// 设置命令模拟
	cleanup := mockCommand(func(name string, args ...string) *exec.Cmd {
		switch name {
		case "systemd-inhibit":
			return exec.Command("printf", "%s", mockSystemdInhibitOutput)
		case "loginctl":
			return exec.Command("printf", "%s", mockLoginctlOutput)
		default:
			return exec.Command("echo", "command not mocked")
		}
	})
	defer cleanup()

	processes, powerState, _, err := system.GetPreventSleepProcesses()
	if err != nil {
		t.Fatalf("GetPreventSleepProcesses failed: %v", err)
	}

	// 验证进程数量（3个 systemd-inhibit 进程 + 1个 login session）
	expectedCount := 4
	if len(processes) != expectedCount {
		t.Errorf("Expected %d processes, got %d", expectedCount, len(processes))
	}

	// 验证进程类型
	expectedTypes := map[string]bool{
		"PreventSystemSleep":         false,
		"PreventUserIdleSystemSleep": false,
	}

	for _, p := range processes {
		expectedTypes[p.Type] = true
		if p.Details == "" {
			t.Errorf("Process %s has empty details", p.Name)
		}

		// 验证详细信息的完整性
		details := system.GetProcessDetailInfo(p)
		if details == "" {
			t.Errorf("Process %s has empty detailed info", p.Name)
		}

		// 验证进程描述
		desc := system.GetProcessDescription(p)
		if desc == "" {
			t.Errorf("Process %s has empty description", p.Name)
		}
	}

	// 验证是否所有类型都被找到
	for typ, found := range expectedTypes {
		if !found {
			t.Errorf("Expected type %s not found", typ)
		}
	}

	// 验证特定进程
	for _, p := range processes {
		switch p.Name {
		case "firefox":
			if p.Type != "PreventUserIdleSystemSleep" {
				t.Errorf("Expected firefox to prevent user idle sleep")
			}
		case "systemd":
			if p.Type != "PreventSystemSleep" {
				t.Errorf("Expected systemd to prevent system sleep")
			}
		}
	}

	// 验证电源状态
	if powerState == nil {
		t.Error("Expected non-nil power state")
	}
}
