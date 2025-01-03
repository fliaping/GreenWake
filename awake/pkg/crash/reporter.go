package crash

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"awake/pkg/logger"
)

type Reporter struct {
	appName    string
	reportDir  string
	maxReports int
}

func NewReporter(appName, reportDir string, maxReports int) *Reporter {
	return &Reporter{
		appName:    appName,
		reportDir:  reportDir,
		maxReports: maxReports,
	}
}

func (r *Reporter) Init() {
	if err := os.MkdirAll(r.reportDir, 0755); err != nil {
		logger.Error("创建崩溃报告目录失败: %v", err)
		return
	}

	// 设置全局 panic 处理器
	defer func() {
		if err := recover(); err != nil {
			r.Report(err)
		}
	}()
}

func (r *Reporter) Report(err interface{}) {
	reportPath := filepath.Join(r.reportDir,
		fmt.Sprintf("crash_%s_%s.log",
			r.appName,
			time.Now().Format("20060102_150405")))

	file, err := os.Create(reportPath)
	if err != nil {
		logger.Error("创建崩溃报告文件失败: %v", err)
		return
	}
	defer file.Close()

	// 写入基本信息
	fmt.Fprintf(file, "时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "系统: %s\n", runtime.GOOS)
	fmt.Fprintf(file, "架构: %s\n", runtime.GOARCH)
	fmt.Fprintf(file, "Go版本: %s\n", runtime.Version())
	fmt.Fprintf(file, "\n错误信息:\n%v\n", err)

	// 写入堆栈信息
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, true)
	fmt.Fprintf(file, "\n堆栈信息:\n%s", buf[:n])

	// 清理旧报告
	r.cleanup()
}

func (r *Reporter) cleanup() {
	entries, err := os.ReadDir(r.reportDir)
	if err != nil {
		return
	}

	if len(entries) <= r.maxReports {
		return
	}

	// 按修改时间排序
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(r.reportDir, entry.Name()))
		}
	}

	// 删除最旧的文件
	for i := 0; i < len(files)-r.maxReports; i++ {
		os.Remove(files[i])
	}
}
