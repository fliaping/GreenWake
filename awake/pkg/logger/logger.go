package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var (
	logFile *os.File
	Logger  *log.Logger
)

func GetLogPath() string {
	var logDir string
	switch runtime.GOOS {
	case "windows":
		logDir = filepath.Join(os.Getenv("APPDATA"), "awake", "logs")
	case "darwin":
		logDir = filepath.Join(os.Getenv("HOME"), "Library", "Logs", "awake")
	case "linux":
		logDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "awake", "logs")
	default:
		logDir = "logs"
	}
	return logDir
}

func Init() error {
	logDir := GetLogPath()
	
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 创建日志文件
	logPath := filepath.Join(logDir, fmt.Sprintf("awake_%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	logFile = file
	
	// 同时输出到文件和终端
	multiWriter := io.MultiWriter(file, os.Stdout)
	Logger = log.New(multiWriter, "", log.LstdFlags)

	Info("日志初始化完成，日志路径: %s", logPath)
	return nil
}

func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

func Info(format string, v ...interface{}) {
	if Logger != nil {
		Logger.Printf("[INFO] "+format, v...)
	}
}

func Error(format string, v ...interface{}) {
	if Logger != nil {
		Logger.Printf("[ERROR] "+format, v...)
	}
}

func Debug(format string, v ...interface{}) {
	if Logger != nil {
		Logger.Printf("[DEBUG] "+format, v...)
	}
}
