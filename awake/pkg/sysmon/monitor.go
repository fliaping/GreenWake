package sysmon

import (
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type Status struct {
	CPUUsage    float64
	MemoryUsage float64
}

type Monitor struct {
	interval time.Duration
	stopCh   chan struct{}
	status   Status
}

func NewMonitor(interval time.Duration) *Monitor {
	return &Monitor{
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (m *Monitor) Start() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.updateStatus()
		case <-m.stopCh:
			return
		}
	}
}

func (m *Monitor) Stop() {
	close(m.stopCh)
}

func (m *Monitor) GetStatus() Status {
	return m.status
}

func (m *Monitor) updateStatus() {
	// 更新 CPU 使用率
	cpuPercent, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercent) > 0 {
		m.status.CPUUsage = cpuPercent[0]
	}

	// 更新内存使用率
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		m.status.MemoryUsage = memInfo.UsedPercent
	}
}
