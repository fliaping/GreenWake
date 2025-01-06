package wakelock

// Lock 定义了系统唤醒锁的接口
type Lock interface {
	// Acquire 获取唤醒锁，阻止系统进入睡眠状态
	Acquire()

	// Release 释放唤醒锁，允许系统进入睡眠状态
	Release()

	// ForceSleep 强制系统进入睡眠状态
	ForceSleep() error
}
