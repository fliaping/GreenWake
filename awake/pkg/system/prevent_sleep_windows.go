//go:build windows
// +build windows

package system

import (
	"awake/pkg/logger"
	"os"
	"os/exec"
	"strings"
)

type windowsProvider struct {
	currentProcess *PreventSleepProcess
}

func newPlatformProvider() PowerStateProvider {
	return &windowsProvider{}
}

func (p *windowsProvider) GetPreventSleepProcesses() ([]PreventSleepProcess, *SystemPowerState, []KernelPowerAssertion, error) {
	// 使用 powercfg /requests 命令获取阻止休眠的进程信息
	cmd := exec.Command("powercfg", "/requests")
	output, err := cmd.Output()
	if err != nil {
		logger.Error("执行 powercfg 命令失败: %v", err)
		return nil, nil, nil, err
	}

	processes := make([]PreventSleepProcess, 0)
	powerState := &SystemPowerState{}
	lines := strings.Split(string(output), "\n")

	var currentProcess PreventSleepProcess
	var currentType string
	var originalInfo strings.Builder

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// 检查是否是新的类型段落
		if strings.HasPrefix(line, "DISPLAY:") {
			currentType = "PreventDisplaySleep"
			originalInfo.Reset()
			originalInfo.WriteString(line)
			continue
		} else if strings.HasPrefix(line, "SYSTEM:") {
			currentType = "PreventSystemSleep"
			originalInfo.Reset()
			originalInfo.WriteString(line)
			continue
		} else if strings.HasPrefix(line, "AWAYMODE:") {
			currentType = "PreventUserIdleSystemSleep"
			originalInfo.Reset()
			originalInfo.WriteString(line)
			continue
		}

		// 解析进程信息
		if strings.HasPrefix(line, "[PROCESS]") || strings.HasPrefix(line, "[DRIVER]") || strings.HasPrefix(line, "[SERVICE]") {
			// 保存之前的进程信息
			if currentProcess.Name != "" {
				processes = append(processes, currentProcess)
			}

			// 创建新的进程信息
			currentProcess = PreventSleepProcess{
				Type: currentType,
			}

			if strings.HasPrefix(line, "[PROCESS]") {
				// 提取进程名
				parts := strings.SplitN(line, "\\", -1)
				if len(parts) > 0 {
					currentProcess.Name = parts[len(parts)-1]
				}
			} else {
				currentProcess.Name = line
			}

			// 保存原始信息
			if originalInfo.Len() > 0 {
				currentProcess.Details = originalInfo.String() + " | " + line
			} else {
				currentProcess.Details = line
			}
		} else if currentProcess.Name != "" {
			// 添加额外的详细信息
			currentProcess.Details += " | " + line
			if currentProcess.Reason == "" {
				currentProcess.Reason = line
			}
		}
	}

	// 添加最后一个进程
	if currentProcess.Name != "" {
		processes = append(processes, currentProcess)
	}

	// 检查当前进程是否在阻止休眠
	for i, proc := range processes {
		if proc.Name == os.Args[0] {
			p.currentProcess = &processes[i]
			break
		}
	}

	return processes, powerState, nil, nil
}

func (p *windowsProvider) GetProcessDescription(process PreventSleepProcess) string {
	switch process.Type {
	case "PreventDisplaySleep":
		return "防止显示器休眠"
	case "PreventSystemSleep":
		return "防止系统休眠"
	case "PreventUserIdleSystemSleep":
		return "防止系统自动休眠"
	default:
		return "未知类型"
	}
}

func (p *windowsProvider) GetPowerStateDescription(state *SystemPowerState) map[string]string {
	return map[string]string{
		"PreventSystemSleep":  "系统休眠阻止",
		"PreventUserIdle":     "用户空闲休眠阻止",
		"PreventDisplaySleep": "显示器休眠阻止",
		"BackgroundActivity":  "后台活动",
		"ExternalDevice":      "外部设备活动",
		"NetworkActivity":     "网络活动",
	}
}

func (p *windowsProvider) GetCurrentProcessState() *PreventSleepProcess {
	return p.currentProcess
}

func (p *windowsProvider) GetProcessDetailInfo(process PreventSleepProcess) string {
	if process.Name == "当前进程" {
		return process.Details
	}

	if process.Details != "" {
		// 清理并格式化详细信息，但保留原始内容的完整性
		details := strings.ReplaceAll(process.Details, "\r\n", " | ")
		details = strings.Join(strings.Fields(details), " ")
		return details
	}

	if process.Reason != "" {
		return process.Reason
	}

	return "无详细信息"
}
