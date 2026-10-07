package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipeDeliversPCM(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "tone.ogg")
	cmd := exec.Command(
		"ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-c:a", "libopus", "-b:a", "64k", out,
		"-f", "s16le", "-ac", "1", "-ar", "16000", "pipe:3",
	)
	cmd.ExtraFiles = []*os.File{write}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	write.Close()
	got, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	if len(got) < 16000 {
		t.Fatalf("pipe:3 delivered %d bytes, want at least 1 second of PCM", len(got))
	}
}

func TestTranscribeLoudRecording(t *testing.T) {
	if _, _, err := locateWhisper(); err != nil {
		t.Skip(err)
	}
	raw := filepath.Join(t.TempDir(), "clip.raw")
	src := "/home/dell/Downloads/meet-2026-10-06-173029.ogg"
	if _, err := os.Stat(src); err != nil {
		t.Skip(err)
	}
	if err := exec.Command(
		"ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-i", src, "-t", "8", "-f", "s16le", "-ac", "1", "-ar", "16000", raw,
	).Run(); err != nil {
		t.Fatal(err)
	}
	pcm, err := os.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	defer pcm.Close()

	client, err := startWhisper()
	if err != nil {
		t.Fatal(err)
	}
	defer client.close()

	textPath := filepath.Join(t.TempDir(), "out.txt")
	textFile, err := os.Create(textPath)
	if err != nil {
		t.Fatal(err)
	}
	pub := &stdoutPublisher{}
	pub.enable()
	if err := transcribePCM(pcm, textFile, pub, client); err != nil {
		t.Fatal(err)
	}
	textFile.Close()
	body, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("transcript: %q", string(body))
	text := strings.TrimSpace(string(body))
	if text == "" {
		t.Fatal("loud recording produced an empty transcript")
	}
	if tooRepetitive(text) {
		t.Fatalf("transcript collapsed into a repeated word: %q", text)
	}
}

func TestWhisperSoftwareTerms(t *testing.T) {
	if _, err := exec.LookPath("espeak-ng"); err != nil {
		t.Skip("espeak-ng is not installed")
	}
	if _, _, err := locateWhisper(); err != nil {
		t.Skip(err)
	}

	dir := t.TempDir()
	wav := filepath.Join(dir, "talk.wav")
	raw := filepath.Join(dir, "talk.raw")
	phrase := "The pull request updates the Kubernetes deployment and the TypeScript API."
	if err := exec.Command("espeak-ng", "-s", "140", "-w", wav, phrase).Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command(
		"ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-i", wav, "-f", "s16le", "-ac", "1", "-ar", "16000", raw,
	).Run(); err != nil {
		t.Fatal(err)
	}
	pcm, err := os.ReadFile(raw)
	if err != nil {
		t.Fatal(err)
	}

	client, err := startWhisper()
	if err != nil {
		t.Fatal(err)
	}
	defer client.close()

	got, err := client.infer(pcm)
	if err != nil {
		t.Fatal(err)
	}
	got = cleanHyp(got)
	t.Logf("hypothesis: %q", got)
	folded := strings.ToLower(got)
	for _, word := range []string{"pull", "request", "kubernetes", "typescript", "api"} {
		if !strings.Contains(folded, word) {
			t.Errorf("missing %q in %q", word, got)
		}
	}
}
