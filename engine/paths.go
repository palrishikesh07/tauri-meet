package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// downloadsDir follows the XDG user-dirs setting when it exists,
// then falls back to ~/Downloads.
func downloadsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	config := filepath.Join(home, ".config", "user-dirs.dirs")
	raw, err := os.ReadFile(config)
	if err == nil {
		if dir := parseDownloadDir(string(raw), home); dir != "" {
			return dir, nil
		}
	}

	return filepath.Join(home, "Downloads"), nil
}

func parseDownloadDir(config, home string) string {
	for _, line := range strings.Split(config, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "XDG_DOWNLOAD_DIR=") {
			continue
		}
		value := strings.TrimPrefix(line, "XDG_DOWNLOAD_DIR=")
		value = strings.Trim(value, `"`)
		value = strings.ReplaceAll(value, "$HOME", home)
		return value
	}
	return ""
}

func monitorID(sinkName string) string {
	return sinkName + ".monitor"
}

// newRecordingPath picks meet-YYYY-MM-DD-HHMMSS.ogg inside Downloads.
func newRecordingPath(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	stamp := time.Now().Format("2006-01-02-150405")
	for n := 1; n <= 50; n++ {
		name := fmt.Sprintf("meet-%s.ogg", stamp)
		if n > 1 {
			name = fmt.Sprintf("meet-%s-%d.ogg", stamp, n)
		}
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path, nil
		}
	}
	return "", fmt.Errorf("could not choose a free file name in %s", dir)
}
