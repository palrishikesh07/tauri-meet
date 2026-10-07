package main

import (
	"strings"
	"testing"
)

func TestAllowedTranscript(t *testing.T) {
	ok := "/home/someone/Downloads/meet-2026-10-06-120000.txt"
	if err := allowedTranscript(ok); err != nil {
		t.Fatal(err)
	}
	rejected := []string{
		"meet-2026-10-06-120000.txt",
		"/tmp/meet-2026-10-06-120000.txt",
		"/home/someone/Downloads/notes.txt",
		"/home/someone/Downloads/meet-2026-10-06-120000.summary.txt",
		"/home/someone/Downloads/meet-2026-10-06-120000.ogg",
	}
	for _, path := range rejected {
		if err := allowedTranscript(path); err == nil {
			t.Fatalf("accepted %s", path)
		}
	}
}

func TestCleanHypKeepsQuestionMark(t *testing.T) {
	got := cleanHyp("  What is a pull request?  ")
	if !strings.HasSuffix(got, "?") {
		t.Fatalf("question mark was removed: %q", got)
	}
}

func TestReadChatStream(t *testing.T) {
	raw := "data: {\"choices\":[{\"delta\":{\"content\":\"Q: Hi\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"?\"}}]}\n\ndata: [DONE]\n"
	var parts []string
	text, err := readChatStream(strings.NewReader(raw), func(piece string) {
		parts = append(parts, piece)
	})
	if err != nil {
		t.Fatal(err)
	}
	if text != "Q: Hi?" || strings.Join(parts, "") != text {
		t.Fatalf("got %q parts %q", text, parts)
	}
}

func TestSignalTargetAvoidsOwnGroup(t *testing.T) {
	got, err := signalTarget(40, 40, 10)
	if err != nil || got != -40 {
		t.Fatalf("got %d %v", got, err)
	}
	got, err = signalTarget(40, 10, 10)
	if err != nil || got != 40 {
		t.Fatalf("signaled own group: %d %v", got, err)
	}
	if _, err := signalTarget(1, 1, 1); err == nil {
		t.Fatal("accepted pid 1")
	}
}

func TestAnswerQuestionRequiresQuestion(t *testing.T) {
	if _, err := answerQuestion("not a question", "", ""); err == nil {
		t.Fatal("accepted text without a question mark")
	}
}
