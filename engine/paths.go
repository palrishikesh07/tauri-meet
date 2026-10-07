package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func downloadsDir() (string, error) {
	home, err := os.UserHomeDir()

	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "windows":
		downloads := filepath.Join(
			home,
			"Downloads",
		)

		if err := os.MkdirAll(
			downloads,
			0o755,
		); err != nil {
			return "", err
		}

		return downloads, nil

	case "darwin":
		downloads := filepath.Join(
			home,
			"Downloads",
		)

		if err := os.MkdirAll(
			downloads,
			0o755,
		); err != nil {
			return "", err
		}

		return downloads, nil

	default:
		configPath := filepath.Join(
			home,
			".config",
			"user-dirs.dirs",
		)

		if raw, err := os.ReadFile(
			configPath,
		); err == nil {
			if downloads := parseDownloadDir(
				string(raw),
				home,
			); downloads != "" {
				if err := os.MkdirAll(
					downloads,
					0o755,
				); err == nil {
					return downloads, nil
				}
			}
		}

		downloads := filepath.Join(
			home,
			"Downloads",
		)

		if err := os.MkdirAll(
			downloads,
			0o755,
		); err != nil {
			return "", err
		}

		return downloads, nil
	}
}

func parseDownloadDir(
	config string,
	home string,
) string {
	for _, line := range strings.Split(
		config,
		"\n",
	) {
		line = strings.TrimSpace(line)

		if !strings.HasPrefix(
			line,
			"XDG_DOWNLOAD_DIR=",
		) {
			continue
		}

		value := strings.TrimPrefix(
			line,
			"XDG_DOWNLOAD_DIR=",
		)

		value = strings.Trim(
			value,
			`"`,
		)

		value = strings.ReplaceAll(
			value,
			"$HOME",
			home,
		)

		return value
	}

	return ""
}

func monitorID(
	sinkName string,
) string {
	return sinkName + ".monitor"
}

func newRecordingPath(
	dir string,
) (string, error) {
	if err := os.MkdirAll(
		dir,
		0o755,
	); err != nil {
		return "",
			err
	}

	stamp := time.Now().Format(
		"2006-01-02-150405",
	)

	for n := 1; n <= 50; n++ {
		name := fmt.Sprintf(
			"meet-%s.ogg",
			stamp,
		)

		if n > 1 {
			name = fmt.Sprintf(
				"meet-%s-%d.ogg",
				stamp,
				n,
			)
		}

		path := filepath.Join(
			dir,
			name,
		)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path, nil
		}
	}

	return "",
		fmt.Errorf(
			"could not choose a free file name in %s",
			dir,
		)
}