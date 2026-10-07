package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type whisperClient struct {
	cmd  *exec.Cmd
	base string
	logs *bytes.Buffer
	http *http.Client
	done chan struct{}
}

func locateWhisper() (string, string, error) {
	roots := make([]string, 0, 2)
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		roots = append(roots, cwd)
	}
	for _, root := range roots {
		bin := filepath.Join(root, "bin", "whisper-server")
		model := filepath.Join(root, "models", "ggml-small.en.bin")
		if _, err := os.Stat(bin); err != nil {
			continue
		}
		if _, err := os.Stat(model); err != nil {
			continue
		}
		return bin, model, nil
	}
	return "", "", errors.New("Whisper is not installed. Expected engine/bin/whisper-server and engine/models/ggml-small.en.bin")
}

func startWhisper() (*whisperClient, error) {
	bin, model, err := locateWhisper()
	if err != nil {
		return nil, err
	}

	var lastErr error
	for port := 18080; port < 18090; port++ {
		logs := &bytes.Buffer{}
		cmd := exec.Command(
			bin,
			"-m", model,
			"--host", "127.0.0.1",
			"--port", strconv.Itoa(port),
			"-l", "en",
			"-t", "2",
			"--no-timestamps",
		)
		cmd.Stdout = logs
		cmd.Stderr = logs
		cmd.Env = append(os.Environ(), "LD_LIBRARY_PATH="+filepath.Dir(bin))
		cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("could not start Whisper: %w", err)
		}
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()

		client := &whisperClient{
			cmd:  cmd,
			base: fmt.Sprintf("http://127.0.0.1:%d", port),
			logs: logs,
			http: &http.Client{Timeout: 90 * time.Second},
			done: done,
		}
		if err := client.waitReady(); err != nil {
			client.close()
			lastErr = err
			if strings.Contains(logs.String(), "couldn't bind") {
				continue
			}
			return nil, err
		}
		return client, nil
	}
	if lastErr == nil {
		lastErr = errors.New("could not start Whisper")
	}
	return nil, lastErr
}

func (c *whisperClient) waitReady() error {
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-c.done:
			msg := strings.TrimSpace(c.logs.String())
			if msg == "" {
				msg = "Whisper stopped while loading the model"
			}
			return errors.New(msg)
		default:
		}
		resp, err := c.http.Get(c.base + "/health")
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK && strings.Contains(string(body), "ok") {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	msg := strings.TrimSpace(c.logs.String())
	if msg == "" {
		msg = "timed out loading the speech model"
	}
	return errors.New(msg)
}

func (c *whisperClient) close() {
	if c == nil || c.cmd == nil || c.cmd.Process == nil {
		return
	}
	_ = c.cmd.Process.Kill()
	<-c.done
}

func (c *whisperClient) infer(pcm []byte) (string, error) {
	wav := encodeWAV(pcm)
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", "phrase.wav")
	if err != nil {
		return "", err
	}
	if _, err := file.Write(wav); err != nil {
		return "", err
	}
	fields := map[string]string{
		"temperature":     "0",
		"temperature_inc": "0.0",
		"response_format": "json",
		"language":        "en",
		"no_timestamps":   "true",
	}
	for key, value := range fields {
		if err := form.WriteField(key, value); err != nil {
			return "", err
		}
	}
	if err := form.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, c.base+"/inference", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Whisper returned %s: %s", resp.Status, strings.TrimSpace(string(payload)))
	}
	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return "", err
	}
	return parsed.Text, nil
}

func encodeWAV(pcm []byte) []byte {
	var buf bytes.Buffer
	dataLen := uint32(len(pcm))
	_ = binary.Write(&buf, binary.LittleEndian, []byte("RIFF"))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(36)+dataLen)
	_ = binary.Write(&buf, binary.LittleEndian, []byte("WAVEfmt "))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16000))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(32000))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(16))
	_ = binary.Write(&buf, binary.LittleEndian, []byte("data"))
	_ = binary.Write(&buf, binary.LittleEndian, dataLen)
	_, _ = buf.Write(pcm)
	return buf.Bytes()
}
