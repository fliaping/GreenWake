package updater

import (
	"encoding/json"
	"io"
	"net/http"
	"runtime"
	"time"

	"awake/pkg/logger"
)

type Version struct {
	Version     string    `json:"version"`
	ReleaseDate time.Time `json:"release_date"`
	Downloads   struct {
		Windows string `json:"windows"`
		MacOS   string `json:"macos"`
		Linux   string `json:"linux"`
	} `json:"downloads"`
}

type Updater struct {
	currentVersion string
	updateURL      string
	checkInterval  time.Duration
	onNewVersion   func(Version)
}

func NewUpdater(currentVersion, updateURL string, checkInterval time.Duration) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		updateURL:      updateURL,
		checkInterval:  checkInterval,
	}
}

func (u *Updater) Start() {
	ticker := time.NewTicker(u.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if version, err := u.checkUpdate(); err == nil {
				if version.Version > u.currentVersion && u.onNewVersion != nil {
					u.onNewVersion(*version)
				}
			} else {
				logger.Error("检查更新失败: %v", err)
			}
		}
	}
}

func (u *Updater) checkUpdate() (*Version, error) {
	resp, err := http.Get(u.updateURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var version Version
	if err := json.Unmarshal(data, &version); err != nil {
		return nil, err
	}

	return &version, nil
}

func (u *Updater) GetDownloadURL(version Version) string {
	switch runtime.GOOS {
	case "windows":
		return version.Downloads.Windows
	case "darwin":
		return version.Downloads.MacOS
	default:
		return version.Downloads.Linux
	}
}
