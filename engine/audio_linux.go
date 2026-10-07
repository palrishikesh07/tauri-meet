
//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// Device is one playback output. Recording uses its ".monitor" source,
// which is a copy of audio heading to the speakers or headphones.
// That is the meeting (what you hear). It is not the microphone.
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

type sinkInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func listDevices() (DeviceList, error) {
	dir, err := downloadsDir()
	if err != nil {
		return DeviceList{}, err
	}

	if _, err := exec.LookPath("pactl"); err != nil {
		return DeviceList{}, errors.New("pactl was not found. Install PipeWire or PulseAudio")
	}

	out, err := exec.Command("pactl", "-f", "json", "list", "sinks").Output()
	if err != nil {
		return DeviceList{}, fmt.Errorf("could not list playback devices: %w", err)
	}

	var sinks []sinkInfo
	if err := json.Unmarshal(out, &sinks); err != nil {
		return DeviceList{}, fmt.Errorf("could not read playback devices: %w", err)
	}

	defaultSink := strings.TrimSpace(commandOutput("pactl", "get-default-sink"))
	devices := make([]Device, 0, len(sinks))
	defaultSource := ""

	for _, sink := range sinks {
		if sink.Name == "" {
			continue
		}
		id := monitorID(sink.Name)
		isDefault := sink.Name == defaultSink
		if isDefault {
			defaultSource = id
		}
		label := sink.Description
		if label == "" {
			label = sink.Name
		}
		devices = append(devices, Device{
			ID:        id,
			Label:     label,
			Detail:    "Playback monitor — the audio this output is playing",
			IsDefault: isDefault,
		})
	}

	sort.SliceStable(devices, func(i, j int) bool {
		if devices[i].IsDefault != devices[j].IsDefault {
			return devices[i].IsDefault
		}
		return devices[i].Label < devices[j].Label
	})

	if defaultSource == "" && len(devices) > 0 {
		defaultSource = devices[0].ID
		devices[0].IsDefault = true
	}

	return DeviceList{
		DownloadsDir:  dir,
		DefaultSource: defaultSource,
		Devices:       devices,
	}, nil
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
		if !strings.HasPrefix(name, "meet-") || !strings.HasSuffix(name, ".ogg") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		audioPath := filepath.Join(dir, name)
		textPath := strings.TrimSuffix(audioPath, ".ogg") + ".txt"
		if _, err := os.Stat(textPath); err != nil {
			textPath = ""
		}
		summaryPath := strings.TrimSuffix(audioPath, ".ogg") + ".summary.txt"
		if _, err := os.Stat(summaryPath); err != nil {
			summaryPath = ""
		}
		files = append(files, RecordingFile{
			Name:        name,
			Path:        audioPath,
			Bytes:       info.Size(),
			Modified:    info.ModTime().UTC().Format(time.RFC3339),
			TextPath:    textPath,
			SummaryPath: summaryPath,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Modified > files[j].Modified
	})
	return files, nil
}

// record captures one playback monitor until this process receives SIGINT or SIGTERM.
// The desktop app stops a take by sending that signal.
func record(source string) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return errors.New("ffmpeg was not found")
	}

	if source == "" {
		devices, err := listDevices()
		if err != nil {
			return err
		}
		source = devices.DefaultSource
		if source == "" {
			return errors.New("no playback device to record")
		}
	}
	if strings.ContainsAny(source, "\n\r") {
		return errors.New("invalid playback source")
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
	textFile, err := os.Create(textPath)
	if err != nil {
		return err
	}
	defer textFile.Close()

	client, err := startWhisper()
	if err != nil {
		return err
	}
	defer client.close()

	pcmRead, pcmWrite, err := os.Pipe()
	if err != nil {
		return err
	}

	// The ogg file is what you hear. pipe:3 is 16 kHz mono PCM for the recognizer.
	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-hide_banner",
		"-loglevel", "error",
		"-f", "pulse",
		"-i", source,
		"-c:a", "libopus",
		"-b:a", "64k",
		"-application", "voip",
		"-vbr", "on",
		output,
		"-flush_packets", "1",
		"-f", "s16le",
		"-ac", "1",
		"-ar", "16000",
		"pipe:3",
	)
	cmd.Stdin = nil
	cmd.ExtraFiles = []*os.File{pcmWrite}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		// If this process is killed, ffmpeg still gets SIGINT and closes the file.
		Pdeathsig: syscall.SIGINT,
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		pcmRead.Close()
		pcmWrite.Close()
		return fmt.Errorf("could not start ffmpeg: %w", err)
	}
	pcmWrite.Close()

	pub := &stdoutPublisher{}
	transcriptDone := make(chan struct{})
	go func() {
		defer close(transcriptDone)
		defer pcmRead.Close()
		_ = transcribePCM(pcmRead, textFile, pub, client)
	}()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		<-transcriptDone
		msg := strings.TrimSpace(stderr.String())
		if msg == "" && err != nil {
			msg = err.Error()
		}
		if msg == "" {
			msg = "ffmpeg stopped before recording started"
		}
		return errors.New(msg)
	case <-time.After(400 * time.Millisecond):
	}

	ready := readyMessage{Status: "recording", Path: output, Source: source, TextPath: textPath}
	if err := writeReady(ready); err != nil {
		_ = stopFFmpeg(cmd)
		<-done
		<-transcriptDone
		return err
	}
	pub.enable()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-done:
	case <-sigs:
		if err := stopFFmpeg(cmd); err != nil {
			<-done
			<-transcriptDone
			return err
		}
		<-done
	}
	<-transcriptDone
	return nil
}

func stopFFmpeg(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return errors.New("ffmpeg is not running")
	}
	target, err := signalTarget(cmd.Process.Pid, processGroup(cmd.Process.Pid), processGroup(os.Getpid()))
	if err != nil {
		return err
	}
	err = syscall.Kill(target, syscall.SIGINT)
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

// signalTarget never returns this process's group. A negative pid would
// signal every process in that group, including the desktop app.
func signalTarget(pid, pgid, selfPgid int) (int, error) {
	if pid <= 1 {
		return 0, errors.New("ffmpeg is not running")
	}
	if pgid <= 1 || pgid == selfPgid {
		return pid, nil
	}
	return -pgid, nil
}

func processGroup(pid int) int {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return 0
	}
	return pgid
}

func commandOutput(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}
