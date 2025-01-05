package system

// PreventSleepProcess 表示阻止系统休眠的进程信息
type PreventSleepProcess struct {
	PID      int    // 进程ID
	Name     string // 进程名
	Reason   string // 阻止休眠的原因
	Type     string // 阻止休眠的类型
	Details  string // 详细信息
	Duration string // 持续时间
}

// SystemPowerState 表示系统电源管理状态
type SystemPowerState struct {
	PreventSystemSleep  bool // 阻止系统休眠
	PreventUserIdle     bool // 阻止用户空闲休眠
	PreventDisplaySleep bool // 阻止显示器休眠
	BackgroundActivity  bool // 后台活动
	ExternalDevice      bool // 外部设备活动
	NetworkActivity     bool // 网络活动
}

// KernelPowerAssertion 表示内核级电源管理断言
type KernelPowerAssertion struct {
	ID          int
	Level       int
	Type        string
	Description string
	Owner       string
	CreateTime  string
	ModTime     string
}

// PowerStateProvider 提供系统电源状态信息的接口
type PowerStateProvider interface {
	// GetPreventSleepProcesses 获取当前阻止系统休眠的进程列表
	GetPreventSleepProcesses() ([]PreventSleepProcess, *SystemPowerState, []KernelPowerAssertion, error)

	// GetProcessDescription 获取进程的本地化描述
	GetProcessDescription(process PreventSleepProcess) string

	// GetPowerStateDescription 获取电源状态的本地化描述
	GetPowerStateDescription(state *SystemPowerState) map[string]string

	// GetCurrentProcessState 获取当前进程的休眠阻止状态
	GetCurrentProcessState() *PreventSleepProcess

	// GetProcessDetailInfo 获取进程的详细信息描述
	GetProcessDetailInfo(process PreventSleepProcess) string
}

var provider PowerStateProvider

// InitPowerStateProvider 初始化电源状态提供者
func InitPowerStateProvider() {
	provider = newPlatformProvider()
}

// GetPreventSleepProcesses 获取当前阻止系统休眠的进程列表
func GetPreventSleepProcesses() ([]PreventSleepProcess, *SystemPowerState, []KernelPowerAssertion, error) {
	if provider == nil {
		InitPowerStateProvider()
	}
	return provider.GetPreventSleepProcesses()
}

// GetProcessDescription 获取进程的本地化描述
func GetProcessDescription(process PreventSleepProcess) string {
	if provider == nil {
		InitPowerStateProvider()
	}
	return provider.GetProcessDescription(process)
}

// GetPowerStateDescription 获取电源状态的本地化描述
func GetPowerStateDescription(state *SystemPowerState) map[string]string {
	if provider == nil {
		InitPowerStateProvider()
	}
	return provider.GetPowerStateDescription(state)
}

// GetCurrentProcessState 获取当前进程的休眠阻止状态
func GetCurrentProcessState() *PreventSleepProcess {
	if provider == nil {
		InitPowerStateProvider()
	}
	return provider.GetCurrentProcessState()
}

// GetProcessDetailInfo 获取进程的详细信息描述
func GetProcessDetailInfo(process PreventSleepProcess) string {
	if provider == nil {
		InitPowerStateProvider()
	}
	return provider.GetProcessDetailInfo(process)
}
