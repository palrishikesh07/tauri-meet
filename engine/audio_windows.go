//go:build windows

package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Device struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Detail    string `json:"detail"`
	IsDefault bool   `json:"isDefault"`
}

type DeviceList struct {
	DownloadsDir  string   `json:"downloadsDir"`
	DefaultSource string   `json:"defaultSource"`
	Devices       []Device `json:"devices"`
}

type RecordingFile struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Bytes       int64  `json:"bytes"`
	Modified    string `json:"modified"`
	TextPath    string `json:"textPath,omitempty"`
	SummaryPath string `json:"summaryPath,omitempty"`
}

type readyMessage struct {
	Status   string `json:"status"`
	Path     string `json:"path"`
	Source   string `json:"source"`
	TextPath string `json:"textPath"`
}

func listDevices() (DeviceList, error) {
	dir, err := downloadsDir()
	if err != nil {
		return DeviceList{}, err
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return DeviceList{}, errors.New(
			"ffmpeg was not found. Install FFmpeg and add it to PATH",
		)
	}

	output, _ := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-list_devices",
		"true",
		"-f",
		"dshow",
		"-i",
		"dummy",
	).CombinedOutput()

	devices := parseDirectShowDevices(
		string(output),
	)

	if len(devices) == 0 {
		return DeviceList{}, errors.New(
			"no Windows playback capture device was found. Enable Stereo Mix / What U Hear in Windows Sound settings",
		)
	}

	devices[0].IsDefault = true

	return DeviceList{
		DownloadsDir:  dir,
		DefaultSource: devices[0].ID,
		Devices:       devices,
	}, nil
}

func parseDirectShowDevices(
	output string,
) []Device {
	var devices []Device

	lines := strings.Split(
		output,
		"\n",
	)

	seen := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if !strings.Contains(line, `"`) {
			continue
		}

		start := strings.Index(
			line,
			`"`,
		)

		end := strings.LastIndex(
			line,
			`"`,
		)

		if start < 0 ||
			end <= start {
			continue
		}

		name := strings.TrimSpace(
			line[start+1 : end],
		)

		if name == "" ||
			seen[name] {
			continue
		}

		lower := strings.ToLower(name)

		// DirectShow lists video and audio devices together.
		// We only want playback-capture devices.
		if !strings.Contains(lower, "stereo mix") &&
			!strings.Contains(lower, "what u hear") &&
			!strings.Contains(lower, "wave out mix") &&
			!strings.Contains(lower, "loopback") {
			continue
		}

		seen[name] = true

		devices = append(
			devices,
			Device{
				ID:     name,
				Label:  name,
				Detail: "Windows playback capture device",
			},
		)
	}

	sort.SliceStable(
		devices,
		func(i, j int) bool {
			return devices[i].Label < devices[j].Label
		},
	)

	return devices
}

func listRecordings() ([]RecordingFile, error) {
	dir, err := downloadsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)

	if err != nil {
		if os.IsNotExist(err) {
			return []RecordingFile{}, nil
		}

		return nil, err
	}

	files := make([]RecordingFile, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		if !strings.HasPrefix(name, "meet-") ||
			!strings.HasSuffix(name, ".ogg") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		audioPath := filepath.Join(
			dir,
			name,
		)

		textPath := strings.TrimSuffix(
			audioPath,
			".ogg",
		) + ".txt"

		if _, err := os.Stat(textPath); err != nil {
			textPath = ""
		}

		summaryPath := strings.TrimSuffix(
			audioPath,
			".ogg",
		) + ".summary.txt"

		if _, err := os.Stat(summaryPath); err != nil {
			summaryPath = ""
		}

		files = append(
			files,
			RecordingFile{
				Name:        name,
				Path:        audioPath,
				Bytes:       info.Size(),
				Modified:    info.ModTime().UTC().Format(time.RFC3339),
				TextPath:    textPath,
				SummaryPath: summaryPath,
			},
		)
	}

	sort.Slice(
		files,
		func(i, j int) bool {
			return files[i].Modified > files[j].Modified
		},
	)

	return files, nil
}

func record(source string) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return errors.New(
			"ffmpeg was not found. Install FFmpeg and add it to PATH",
		)
	}

	if source == "" {
		devices, err := listDevices()
		if err != nil {
			return err
		}

		source = devices.DefaultSource
		if source == "" {
			return errors.New(
				"no Windows playback capture device is available",
			)
		}
	}

	if strings.ContainsAny(source, "\r\n") {
		return errors.New("invalid audio source")
	}

	dir, err := downloadsDir()
	if err != nil {
		return err
	}

	output, err := newRecordingPath(dir)
	if err != nil {
		return err
	}

	textPath := strings.TrimSuffix(output, ".ogg") + ".txt"

	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-hide_banner",
		"-loglevel",
		"error",
		"-f",
		"dshow",
		"-i",
		"audio="+source,
		"-c:a",
		"libopus",
		"-b:a",
		"64k",
		"-application",
		"voip",
		"-vbr",
		"on",
		"-draw_mouse",
		"0",
		output,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// FFmpeg can be stopped cleanly by sending `q` to stdin. This lets the
	// OGG muxer finish the file instead of killing FFmpeg mid-write.
	ffmpegStdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("could not open ffmpeg stdin: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = ffmpegStdin.Close()
		return fmt.Errorf("could not start ffmpeg: %w", err)
	}

	ready := readyMessage{
		Status:   "recording",
		Path:     output,
		Source:   source,
		TextPath: textPath,
	}

	if err := writeReady(ready); err != nil {
		_, _ = ffmpegStdin.Write([]byte("q\n"))
		_ = ffmpegStdin.Close()
		_ = cmd.Wait()
		return err
	}

	// Tauri keeps this process alive while recording. On Windows, Tauri sends
	// "stop" through the recorder's stdin when the user clicks Stop.
	stopRequested := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			if strings.EqualFold(strings.TrimSpace(scanner.Text()), "stop") {
				close(stopRequested)
				return
			}
		}
		// EOF also means the parent asked us to stop.
		close(stopRequested)
	}()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return errors.New(msg)
		}
		return errors.New("ffmpeg stopped before recording was stopped")

	case <-stopRequested:
		// Gracefully stop FFmpeg so the OGG file is finalized correctly.
		_, _ = ffmpegStdin.Write([]byte("q\n"))
		_ = ffmpegStdin.Close()

		if err := <-done; err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return errors.New(msg)
		}
	}

	meta, err := os.Stat(output)
	if err != nil {
		return fmt.Errorf("recording stopped, but the file was not written: %w", err)
	}
	if meta.Size() == 0 {
		return errors.New("recording stopped, but the audio file is empty")
	}

	// Keep the raw OGG and send it directly to OpenAI transcription.
	// This removes local Whisper startup and the OGG -> PCM conversion.
	transcript, err := transcribeAudioFile(output)
	if err != nil {
		return fmt.Errorf("OpenAI transcription failed: %w", err)
	}

	transcript = strings.TrimSpace(transcript)
	if transcript == "" {
		return errors.New("OpenAI returned an empty transcript")
	}

	if err := os.WriteFile(
		textPath,
		[]byte(transcript+"\n"),
		0644,
	); err != nil {
		return fmt.Errorf("could not save transcript: %w", err)
	}

	return nil
}
