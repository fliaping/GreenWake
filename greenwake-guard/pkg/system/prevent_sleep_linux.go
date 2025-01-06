//go:build linux
// +build linux

package system

import (
	"fmt"
	"greenwake-guard/pkg/logger"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type linuxProvider struct {
	currentProcess *PreventSleepProcess
}

func newPlatformProvider() PowerStateProvider {
	return &linuxProvider{}
}

func (p *linuxProvider) GetPreventSleepProcesses() ([]PreventSleepProcess, *SystemPowerState, []KernelPowerAssertion, error) {
	processes := make([]PreventSleepProcess, 0)
	powerState := &SystemPowerState{}

	// 首先尝试使用 systemd-inhibit
	if procs, err := p.getSystemdInhibitors(); err == nil {
		processes = append(processes, procs...)
	} else {
		logger.Debug("systemd-inhibit 不可用: %v", err)
	}

	// 检查 logind 会话状态
	if procs, err := p.getLoginSessionInhibitors(); err == nil {
		processes = append(processes, procs...)
	} else {
		logger.Debug("logind 会话检查失败: %v", err)
	}

	// 如果是桌面环境，尝试获取桌面相关的状态
	if p.isDesktopEnvironment() {
		if procs, err := p.getDesktopInhibitors(); err == nil {
			processes = append(processes, procs...)
		}
	}

	return processes, powerState, nil, nil
}

func (p *linuxProvider) getSystemdInhibitors() ([]PreventSleepProcess, error) {
	cmd := exec.Command("systemd-inhibit", "--list", "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	processes := make([]PreventSleepProcess, 0)
	lines := strings.Split(string(output), "\n")
	var originalInfo strings.Builder

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "WHO") {
			continue
		}

		// systemd-inhibit 输出格式：
		// WHO         UID  PID  COMM      WHAT     WHY                                                      MODE
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		pid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}

		what := fields[4]
		why := strings.Join(fields[6:], " ")

		// 确定阻止休眠的类型
		var processType string
		switch what {
		case "sleep":
			processType = "PreventSystemSleep"
		case "idle":
			processType = "PreventUserIdleSystemSleep"
		case "handle-power-key", "handle-suspend-key", "handle-hibernate-key":
			processType = "PreventSystemSleep"
		case "handle-lid-switch":
			processType = "PreventDisplaySleep"
		default:
			processType = "PreventUserIdleSystemSleep"
		}

		// 保存原始信息
		originalInfo.WriteString(line)

		process := PreventSleepProcess{
			PID:     pid,
			Name:    fields[3],
			Type:    processType,
			Reason:  why,
			Details: originalInfo.String(),
		}

		// 获取进程的更多信息
		if cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid)); err == nil {
			process.Details += " | " + string(cmdline)
		}

		processes = append(processes, process)

		// 检查是否是当前进程
		if pid == os.Getpid() {
			p.currentProcess = &process
		}

		originalInfo.Reset()
	}

	return processes, nil
}

func (p *linuxProvider) getLoginSessionInhibitors() ([]PreventSleepProcess, error) {
	cmd := exec.Command("loginctl", "show-session")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	processes := make([]PreventSleepProcess, 0)
	lines := strings.Split(string(output), "\n")
	var details strings.Builder

	for _, line := range lines {
		if strings.Contains(line, "IdleHint=no") {
			details.WriteString(line)
			processes = append(processes, PreventSleepProcess{
				Name:    "Login Session",
				Type:    "PreventUserIdleSystemSleep",
				Reason:  "用户会话活动",
				Details: details.String(),
			})
			break
		}
		details.WriteString(line)
		details.WriteString(" | ")
	}

	return processes, nil
}

func (p *linuxProvider) isDesktopEnvironment() bool {
	// 检查是否存在显示服务器
	if _, err := os.Stat("/tmp/.X11-unix"); err == nil {
		return true
	}
	// 检查是否存在 Wayland 显示服务器
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	return false
}

func (p *linuxProvider) getDesktopInhibitors() ([]PreventSleepProcess, error) {
	processes := make([]PreventSleepProcess, 0)
	var details strings.Builder

	// 检查 GNOME session（如果可用）
	if out, err := exec.Command("gsettings", "get", "org.gnome.desktop.session", "idle-delay").Output(); err == nil {
		if strings.TrimSpace(string(out)) == "uint32 0" {
			details.WriteString("GNOME Session settings: idle-delay=0")
			processes = append(processes, PreventSleepProcess{
				Name:    "GNOME Session",
				Type:    "PreventUserIdleSystemSleep",
				Reason:  "用户禁用了空闲休眠",
				Details: details.String(),
			})
		}
	}

	details.Reset()

	// 检查 Xorg DPMS（如果可用）
	if out, err := exec.Command("xset", "q").Output(); err == nil {
		details.WriteString(string(out))
		if strings.Contains(string(out), "DPMS is Disabled") {
			processes = append(processes, PreventSleepProcess{
				Name:    "X11 DPMS",
				Type:    "PreventDisplaySleep",
				Reason:  "显示器电源管理已禁用",
				Details: details.String(),
			})
		}
	}

	return processes, nil
}

func (p *linuxProvider) GetProcessDescription(process PreventSleepProcess) string {
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

func (p *linuxProvider) GetPowerStateDescription(state *SystemPowerState) map[string]string {
	return map[string]string{
		"PreventSystemSleep":  "系统休眠阻止",
		"PreventUserIdle":     "用户空闲休眠阻止",
		"PreventDisplaySleep": "显示器休眠阻止",
		"BackgroundActivity":  "后台活动",
		"ExternalDevice":      "外部设备活动",
		"NetworkActivity":     "网络活动",
	}
}

func (p *linuxProvider) GetCurrentProcessState() *PreventSleepProcess {
	return p.currentProcess
}

func (p *linuxProvider) GetProcessDetailInfo(process PreventSleepProcess) string {
	if process.Name == "当前进程" {
		return process.Details
	}

	if process.Details != "" {
		// 清理并格式化详细信息，但保留原始内容的完整性
		details := strings.ReplaceAll(process.Details, "\n", " | ")
		details = strings.Join(strings.Fields(details), " ")
		return details
	}

	if process.Reason != "" {
		return process.Reason
	}

	return "无详细信息"
}
