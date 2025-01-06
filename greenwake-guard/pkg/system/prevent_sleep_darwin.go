//go:build darwin
// +build darwin

package system

import (
	"greenwake-guard/pkg/logger"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// darwinPreventSleepProcess 表示 macOS 平台特有的进程信息
type darwinPreventSleepProcess struct {
	PreventSleepProcess        // 嵌入通用进程信息
	Timeout             string // 超时时间（macOS 特有）
	Action              string // 超时动作（macOS 特有）
	Localized           string // 本地化描述（macOS 特有）
}

type darwinProvider struct {
	currentProcess *PreventSleepProcess
}

func newPlatformProvider() PowerStateProvider {
	return &darwinProvider{}
}

func (p *darwinProvider) GetPreventSleepProcesses() ([]PreventSleepProcess, *SystemPowerState, []KernelPowerAssertion, error) {
	cmd := exec.Command("pmset", "-g", "assertions")
	output, err := cmd.Output()
	if err != nil {
		logger.Error("执行 pmset 命令失败: %v", err)
		return nil, nil, nil, err
	}

	processes := make([]PreventSleepProcess, 0)
	powerState := &SystemPowerState{}
	kernelAssertions := make([]KernelPowerAssertion, 0)

	lines := strings.Split(string(output), "\n")
	section := ""

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// 确定当前处理的部分
		if strings.Contains(line, "Assertion status system-wide:") {
			section = "system"
			continue
		} else if strings.Contains(line, "Listed by owning process:") {
			section = "process"
			continue
		} else if strings.Contains(line, "Kernel Assertions:") {
			section = "kernel"
			continue
		}

		switch section {
		case "system":
			p.parseSystemAssertion(line, powerState)
		case "process":
			if strings.HasPrefix(line, "pid ") {
				process := p.parseProcessLine(line)
				// 获取后续的详细信息
				for i++; i < len(lines); i++ {
					detail := strings.TrimSpace(lines[i])
					if !strings.HasPrefix(detail, "\t") {
						i--
						break
					}
					p.parseProcessDetail(detail, &process)
				}
				processes = append(processes, process)

				// 检查是否是当前进程
				if process.PID == os.Getpid() {
					p.currentProcess = &process
				}
			}
		case "kernel":
			if ka := p.parseKernelAssertion(line); ka != nil {
				kernelAssertions = append(kernelAssertions, *ka)
			}
		}
	}

	return processes, powerState, kernelAssertions, nil
}

func (p *darwinProvider) GetProcessDescription(process PreventSleepProcess) string {
	// 如果是 caffeinate 进程，返回特定描述
	if process.Name == "caffeinate" {
		return "命令行阻止休眠工具"
	}
	return p.getCommonReasonDescription(process.Reason, process.Type)
}

func (p *darwinProvider) GetPowerStateDescription(state *SystemPowerState) map[string]string {
	return map[string]string{
		"PreventSystemSleep":  "系统休眠阻止",
		"PreventUserIdle":     "用户空闲休眠阻止",
		"PreventDisplaySleep": "显示器休眠阻止",
		"BackgroundActivity":  "后台活动",
		"ExternalDevice":      "外部设备活动",
		"NetworkActivity":     "网络活动",
	}
}

func (p *darwinProvider) GetCurrentProcessState() *PreventSleepProcess {
	return p.currentProcess
}

func (p *darwinProvider) parseSystemAssertion(line string, ps *SystemPowerState) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return
	}
	value := parts[len(parts)-1] == "1"
	switch parts[0] {
	case "BackgroundTask":
		ps.BackgroundActivity = value
	case "PreventSystemSleep":
		ps.PreventSystemSleep = value
	case "PreventUserIdleSystemSleep":
		ps.PreventUserIdle = value
	case "PreventUserIdleDisplaySleep", "InternalPreventDisplaySleep":
		ps.PreventDisplaySleep = value
	case "ExternalMedia":
		ps.ExternalDevice = value
	case "NetworkClientActive":
		ps.NetworkActivity = value
	}
}

func (p *darwinProvider) parseProcessLine(line string) PreventSleepProcess {
	process := PreventSleepProcess{}

	// 提取 PID
	if pidStart := strings.Index(line, "pid "); pidStart != -1 {
		pidStr := strings.TrimSpace(line[pidStart+4:])
		if pidEnd := strings.Index(pidStr, "("); pidEnd != -1 {
			if pid, err := strconv.Atoi(pidStr[:pidEnd]); err == nil {
				process.PID = pid
			}
		}
	}

	// 提取进程名
	if start := strings.Index(line, "("); start != -1 {
		if end := strings.Index(line[start:], ")"); end != -1 {
			process.Name = line[start+1 : start+end]
		}
	}

	// 提取持续时间
	if timeStart := strings.Index(line, "] "); timeStart != -1 {
		if timeEnd := strings.Index(line[timeStart+2:], " "); timeEnd != -1 {
			process.Duration = line[timeStart+2 : timeStart+2+timeEnd]
		}
	}

	// 提取原因
	if start := strings.Index(line, "named: \""); start != -1 {
		line = line[start+7:]
		if end := strings.Index(line, "\""); end != -1 {
			process.Reason = line[:end]
		}
	}

	// 根据原始信息判断阻止休眠的类型
	switch {
	case strings.Contains(line, "PreventUserIdleSystemSleep"):
		process.Type = "PreventUserIdleSystemSleep"
	case strings.Contains(line, "PreventSystemSleep"):
		process.Type = "PreventSystemSleep"
	case strings.Contains(line, "PreventDisplaySleep"):
		process.Type = "PreventDisplaySleep"
	case strings.Contains(line, "BackgroundTask"):
		process.Type = "BackgroundTask"
	case strings.Contains(line, "NetworkClientActive"):
		process.Type = "NetworkClientActive"
	case strings.Contains(line, "ExternalMedia"):
		process.Type = "ExternalMedia"
	default:
		// 如果没有明确的类型标识，根据原因和进程特征推断
		if strings.Contains(process.Reason, "display") {
			process.Type = "PreventDisplaySleep"
		} else if strings.Contains(process.Reason, "network") {
			process.Type = "NetworkClientActive"
		} else if strings.Contains(process.Reason, "background") {
			process.Type = "BackgroundTask"
		} else {
			process.Type = "PreventUserIdleSystemSleep"
		}
	}

	// 保存原始命令行输出作为详细信息的开始
	process.Details = line

	return process
}

func (p *darwinProvider) parseProcessDetail(line string, process *PreventSleepProcess) {
	line = strings.TrimPrefix(line, "\t")

	// 直接将每行详细信息追加到 Details 字段
	if process.Details == "" {
		process.Details = line
	} else {
		process.Details += " | " + line
	}
}

func (p *darwinProvider) parseKernelAssertion(line string) *KernelPowerAssertion {
	if !strings.HasPrefix(line, "   id=") {
		return nil
	}

	ka := &KernelPowerAssertion{}
	parts := strings.Fields(line)

	for _, part := range parts {
		if strings.HasPrefix(part, "id=") {
			if id, err := strconv.Atoi(strings.TrimPrefix(part, "id=")); err == nil {
				ka.ID = id
			}
		} else if strings.HasPrefix(part, "level=") {
			if level, err := strconv.Atoi(strings.TrimPrefix(part, "level=")); err == nil {
				ka.Level = level
			}
		} else if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				switch kv[0] {
				case "description":
					ka.Description = kv[1]
				case "owner":
					ka.Owner = kv[1]
				case "creat":
					ka.CreateTime = kv[1]
				case "mod":
					ka.ModTime = kv[1]
				}
			}
		}
	}

	return ka
}

func (p *darwinProvider) getCommonReasonDescription(reason, assertType string) string {
	switch assertType {
	case "PreventUserIdleSystemSleep":
		return "防止系统自动休眠"
	case "PreventSystemSleep":
		return "防止系统强制休眠"
	case "PreventDisplaySleep":
		return "防止显示器休眠"
	case "BackgroundTask":
		return "后台任务"
	case "NetworkClientActive":
		return "网络活动"
	case "ExternalMedia":
		return "外部设备"
	}

	// 如果没有匹配的类型，返回原因的本地化描述
	commonReasons := map[string]string{
		"com.apple.powermanagement.timetowake": "定时唤醒",
		"com.apple.backupd-auto":               "Time Machine 备份",
		"com.apple.backupd":                    "Time Machine 备份",
		"com.apple.Safari.PowerSaveBlocker":    "Safari 播放媒体",
		"com.apple.iTunes.playback":            "iTunes/Music 播放媒体",
		"com.apple.audio.AppleHDAEngineOutput": "音频输出",
		"displayAssertion":                     "显示器保持开启",
		"UserIsActive":                         "用户活动",
		"com.apple.BTStack":                    "蓝牙设备连接",
		"caffeinate command-line tool":         "命令行阻止休眠工具",
	}

	if desc, ok := commonReasons[reason]; ok {
		return desc
	}
	return reason
}

// GetProcessDetailInfo 获取进程的详细信息描述
func (p *darwinProvider) GetProcessDetailInfo(process PreventSleepProcess) string {
	// 如果是当前进程，直接返回 Details
	if process.Name == "当前进程" {
		return process.Details
	}

	// 对于其他进程，返回原始的命令行输出信息
	if process.Details != "" {
		// 清理多余的空格和制表符，但保留原始信息
		details := strings.ReplaceAll(process.Details, "\t", " ")
		details = strings.Join(strings.Fields(details), " ")
		return details
	}

	// 如果没有详细信息，返回原因
	if process.Reason != "" {
		return process.Reason
	}

	return "无详细信息"
}
