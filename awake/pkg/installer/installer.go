package installer

import (
	"runtime"
)

type Installer interface {
	Install() error
	Uninstall() error
	IsInstalled() bool
}

func NewInstaller() Installer {
	switch runtime.GOOS {
	case "darwin":
		return &macInstaller{}
	case "windows":
		return &windowsInstaller{}
	case "linux":
		return &linuxInstaller{}
	default:
		return nil
	}
}
