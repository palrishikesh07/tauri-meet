package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRecognizeSpokenPhrase(t *testing.T) {
	if _, err := exec.LookPath("espeak-ng"); err != nil {
		t.Skip("espeak-ng is not installed")
	}

	dir := t.TempDir()
	wav := filepath.Join(dir, "hello.wav")
	raw := filepath.Join(dir, "hello.raw")
	if err := exec.Command("espeak-ng", "-s", "120", "-w", wav, "good morning").Run(); err != nil {
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
	rec, err := openRecognizer()
	if err != nil {
		t.Fatal(err)
	}
	defer rec.close()
	if err := rec.start(); err != nil {
		t.Fatal(err)
	}
	silence := make([]byte, 16000) // 0.5 s of 16 kHz s16le
	rec.feed(silence)
	rec.feed(pcm)
	rec.feed(silence)
	got := cleanHyp(rec.end())
	t.Logf("hypothesis: %q", got)
	if got == "" {
		t.Fatal("recognizer returned an empty hypothesis")
	}
}
