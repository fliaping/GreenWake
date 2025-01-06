//go:build darwin
// +build darwin

package system_test

import (
	"greenwake-guard/pkg/system"
	"os/exec"
	"strings"
	"testing"
)

// mockPmsetOutput 模拟 pmset -g assertions 命令输出
var mockPmsetOutput = `Assertion status system-wide:
   BackgroundTask                 0
   PreventUserIdleSystemSleep    1
   PreventSystemSleep            0
   PreventDisplaySleep           1
   ExternalMedia                 0
   NetworkClientActive           0

Listed by owning process:
pid 12345(firefox): [0x0000000100000bc3] PreventUserIdleSystemSleep named: "Firefox is playing audio" 
	Details: Firefox is playing media
	Timeout will fire in 3600 secs Action=TimeoutActionRelease
	Localized=Firefox 正在播放媒体
pid 67890(WindowServer): [0x0000000100000bc4] PreventDisplaySleep named: "com.apple.iohideventsystem.queue.tickle serviceID:100181bfe"
	Details: Display is active
pid 11111(caffeinate): [0x0000000100000bc5] PreventUserIdleSystemSleep named: "caffeinate command-line tool"
	Details: User requested prevent sleep

Kernel Assertions: 0x100
   id=100  level=0 0x100 PreventUserIdleSystemSleep "com.apple.powermanagement.wakeschedule" 
	Details: Next maintenance wake [TCPKeepAlive] scheduled for 2024-01-05 03:00:00 +0000`

// TestDarwinPowerAssertions 测试 macOS 平台电源断言信息获取
func TestDarwinPowerAssertions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Darwin power assertions test in short mode")
	}

	// 设置命令模拟
	cleanup := mockCommand(func(name string, args ...string) *exec.Cmd {
		if name == "pmset" && len(args) > 0 && args[0] == "-g" && args[1] == "assertions" {
			// 使用 printf 确保换行符正确处理
			return exec.Command("printf", "%s", mockPmsetOutput)
		}
		return exec.Command("echo", "command not mocked")
	})
	defer cleanup()

	processes, state, assertions, err := system.GetPreventSleepProcesses()
	if err != nil {
		t.Fatalf("GetPreventSleepProcesses failed: %v", err)
	}

	// 验证系统状态
	if state == nil {
		t.Error("Expected non-nil system state")
	} else {
		if !state.PreventUserIdle {
			t.Error("Expected PreventUserIdle to be true")
		}
		if !state.PreventDisplaySleep {
			t.Error("Expected PreventDisplaySleep to be true")
		}
	}

	// 验证进程数量
	expectedProcessCount := 3 // firefox, WindowServer, caffeinate
	if len(processes) != expectedProcessCount {
		t.Errorf("Expected %d processes, got %d", expectedProcessCount, len(processes))
		// 打印实际进程列表以帮助调试
		for _, p := range processes {
			t.Logf("Found process: %s (Type: %s)", p.Name, p.Type)
		}
	}

	// 创建进程映射以便于查找
	processMap := make(map[string]system.PreventSleepProcess)
	for _, p := range processes {
		processMap[p.Name] = p
	}

	// 验证特定进程
	expectedProcesses := map[string]struct {
		expectedType string
		validate     func(p system.PreventSleepProcess) error
	}{
		"firefox": {
			expectedType: "PreventUserIdleSystemSleep",
			validate: func(p system.PreventSleepProcess) error {
				if !strings.Contains(p.Details, "Firefox is playing media") {
					t.Error("Expected firefox details to contain media playback info")
				}
				return nil
			},
		},
		"WindowServer": {
			expectedType: "PreventDisplaySleep",
			validate:     nil,
		},
		"caffeinate": {
			expectedType: "PreventUserIdleSystemSleep",
			validate:     nil,
		},
	}

	for name, expected := range expectedProcesses {
		if p, ok := processMap[name]; ok {
			if p.Type != expected.expectedType {
				t.Errorf("Process %s: expected type %s, got %s", name, expected.expectedType, p.Type)
			}
			if expected.validate != nil {
				expected.validate(p)
			}
		} else {
			t.Errorf("Expected process %s not found", name)
		}
	}

	// 验证内核断言
	if len(assertions) != 1 {
		t.Errorf("Expected 1 kernel assertion, got %d", len(assertions))
		// 打印实际断言列表以帮助调试
		for _, a := range assertions {
			t.Logf("Found assertion: ID=%d, Type=%s", a.ID, a.Type)
		}
	} else {
		a := assertions[0]
		if a.ID != 100 {
			t.Errorf("Expected assertion ID 100, got %d", a.ID)
		}
		if a.Type != "PreventUserIdleSystemSleep" {
			t.Errorf("Expected PreventUserIdleSystemSleep assertion type, got %s", a.Type)
		}
	}
}
