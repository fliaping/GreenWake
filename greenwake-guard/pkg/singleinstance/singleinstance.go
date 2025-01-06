package singleinstance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type ProcessInfo struct {
	Pid       int       `json:"pid"`
	StartTime time.Time `json:"start_time"`
	Config    any       `json:"config"`
}

// LockFile 表示单实例锁文件
type LockFile struct {
	path string
	file *os.File
}

// New 创建新的单实例锁
func New(appName string) *LockFile {
	// 在用户目录下创建锁文件
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}

	lockPath := filepath.Join(homeDir, "Library", "Application Support", appName, fmt.Sprintf("%s.lock", appName))
	return &LockFile{
		path: lockPath,
	}
}

// TryLock 尝试获取锁，如果已经有实例在运行，则返回错误
func (l *LockFile) TryLock(config any) error {
	// 确保锁文件目录存在
	if err := os.MkdirAll(filepath.Dir(l.path), 0755); err != nil {
		return fmt.Errorf("创建锁文件目录失败: %v", err)
	}

	// 尝试创建/打开锁文件
	file, err := os.OpenFile(l.path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		// 如果文件已存在，读取现有进程信息
		content, err := os.ReadFile(l.path)
		if err != nil {
			return fmt.Errorf("读取锁文件失败: %v", err)
		}

		var procInfo ProcessInfo
		if err := json.Unmarshal(content, &procInfo); err != nil {
			return fmt.Errorf("解析锁文件内容失败: %v", err)
		}

		// 检查进程是否还在运行
		if proc, err := os.FindProcess(procInfo.Pid); err == nil {
			// 在Unix系统中，FindProcess总是成功的，需要发送信号0来检查进程是否存在
			if err := proc.Signal(syscall.Signal(0)); err == nil {
				return fmt.Errorf("程序已在运行 - PID: %d, 启动时间: %v, 配置: %+v",
					procInfo.Pid, procInfo.StartTime.Format("2006-01-02 15:04:05"), procInfo.Config)
			}
		}

		// 如果进程不存在，删除旧的锁文件并重试
		os.Remove(l.path)
		return l.TryLock(config)
	}

	// 写入当前进程信息
	procInfo := ProcessInfo{
		Pid:       os.Getpid(),
		StartTime: time.Now(),
		Config:    config,
	}

	data, err := json.MarshalIndent(procInfo, "", "  ")
	if err != nil {
		file.Close()
		os.Remove(l.path)
		return fmt.Errorf("序列化进程信息失败: %v", err)
	}

	if _, err := file.Write(data); err != nil {
		file.Close()
		os.Remove(l.path)
		return fmt.Errorf("写入进程信息失败: %v", err)
	}

	l.file = file
	return nil
}

// Release 释放锁
func (l *LockFile) Release() {
	if l.file != nil {
		l.file.Close()
		os.Remove(l.path)
		l.file = nil
	}
}
