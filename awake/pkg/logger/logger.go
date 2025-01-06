package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelError
)

var (
	logFile  *os.File
	Logger   *log.Logger
	logLevel LogLevel
)

// 将字符串转换为日志级别
func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "error":
		return LevelError
	default:
		return LevelDebug
	}
}

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

func Init(level string) error {
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

	// 设置日志级别
	logLevel = parseLogLevel(level)

	Info("日志初始化完成，日志路径: %s, 日志级别: %s", logPath, level)
	return nil
}

func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

func Info(format string, v ...interface{}) {
	if Logger != nil && logLevel <= LevelInfo {
		Logger.Printf("[INFO] "+format, v...)
	}
}

func Error(format string, v ...interface{}) {
	if Logger != nil && logLevel <= LevelError {
		Logger.Printf("[ERROR] "+format, v...)
	}
}

func Debug(format string, v ...interface{}) {
	if Logger != nil && logLevel <= LevelDebug {
		Logger.Printf("[DEBUG] "+format, v...)
	}
}
