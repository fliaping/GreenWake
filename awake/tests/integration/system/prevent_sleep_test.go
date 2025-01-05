package system_test

import (
	"awake/pkg/system"
	"testing"
)

// TestPreventSleepIntegration 测试阻止休眠功能的基本集成
func TestPreventSleepIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 获取进程信息
	processes, state, assertions, err := system.GetPreventSleepProcesses()
	if err != nil {
		t.Fatalf("Failed to get prevent sleep processes: %v", err)
	}

	// 验证基本信息
	if len(processes) == 0 && state == nil {
		t.Error("Expected either processes or state to be non-empty")
	}

	// 验证每个进程的信息完整性
	for _, p := range processes {
		if p.Name == "" {
			t.Error("Process name should not be empty")
		}
		if p.Type == "" {
			t.Error("Process type should not be empty")
		}
		desc := system.GetProcessDescription(p)
		if desc == "" {
			t.Error("Process description should not be empty")
		}
		details := system.GetProcessDetailInfo(p)
		if details == "" {
			t.Error("Process details should not be empty")
		}
	}

	// 验证电源状态描述
	if state != nil {
		desc := system.GetPowerStateDescription(state)
		if len(desc) == 0 {
			t.Error("Power state description should not be empty")
		}
	}

	// 验证内核断言（如果支持）
	if len(assertions) > 0 {
		for _, a := range assertions {
			if a.Type == "" {
				t.Error("Assertion type should not be empty")
			}
		}
	}
}
