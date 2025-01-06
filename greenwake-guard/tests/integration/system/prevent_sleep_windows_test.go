//go:build windows
// +build windows

package system_test

import (
	"greenwake-guard/pkg/system"
	"os/exec"
	"testing"
)

// mockCommand 模拟 Windows powercfg 命令输出
var mockPowercfgOutput = `DISPLAY:
[PROCESS] \Device\HarddiskVolume4\Program Files\App\example.exe
  Requesting Display availability
[SERVICE] Service1
  Some service description
SYSTEM:
[DRIVER] Driver1
  Some driver description
AWAYMODE:
[PROCESS] \Device\HarddiskVolume4\Program Files\App2\test.exe
  Preventing system sleep`

// TestWindowsPowerRequests 测试 Windows 平台电源请求信息获取
func TestWindowsPowerRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Windows power requests test in short mode")
	}

	// 设置命令模拟
	cleanup := mockCommand(func(name string, args ...string) *exec.Cmd {
		if name == "powercfg" && len(args) > 0 && args[0] == "/requests" {
			return exec.Command("cmd", "/C", "echo "+mockPowercfgOutput)
		}
		return exec.Command("echo", "command not mocked")
	})
	defer cleanup()

	processes, powerState, _, err := system.GetPreventSleepProcesses()
	if err != nil {
		t.Fatalf("GetPreventSleepProcesses failed: %v", err)
	}

	// 验证进程数量
	if len(processes) != 4 {
		t.Errorf("Expected 4 processes, got %d", len(processes))
	}

	// 验证进程类型
	expectedTypes := map[string]bool{
		"PreventDisplaySleep":        false,
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

	// 验证电源状态
	if powerState == nil {
		t.Error("Expected non-nil power state")
	}
}
