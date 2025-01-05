package system_test

import (
	"os/exec"
)

// execCommand 用于模拟命令执行
var execCommand = exec.Command

// mockCommand 设置模拟的命令执行函数
func mockCommand(mockFunc func(string, ...string) *exec.Cmd) func() {
	original := execCommand
	execCommand = mockFunc
	return func() {
		execCommand = original
	}
}
