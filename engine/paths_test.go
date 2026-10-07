package main

import "testing"

func TestMonitorID(t *testing.T) {
	got := monitorID("bluez_output.headphones")
	want := "bluez_output.headphones.monitor"
	if got != want {
		t.Fatalf("monitorID() = %q, want %q", got, want)
	}
}

func TestParseDownloadDir(t *testing.T) {
	config := `
# comment
XDG_DESKTOP_DIR="$HOME/Desktop"
XDG_DOWNLOAD_DIR="$HOME/Downloads"
`
	got := parseDownloadDir(config, "/home/dell")
	if got != "/home/dell/Downloads" {
		t.Fatalf("parseDownloadDir() = %q", got)
	}
}

func TestParseDownloadDirMissing(t *testing.T) {
	if got := parseDownloadDir("XDG_DESKTOP_DIR=\"$HOME/Desktop\"\n", "/home/dell"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
