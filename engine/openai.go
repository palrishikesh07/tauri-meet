package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const openAIModel = "gpt-4o-mini"

type openAIResult struct {
	Text        string `json:"text"`
	SummaryPath string `json:"summaryPath"`
}

type keyStatus struct {
	Saved bool `json:"saved"`
}

func loadDotEnv() {
	for _, path := range dotEnvCandidates() {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.Trim(strings.TrimSpace(value), `"'`)
			if key == "" || os.Getenv(key) != "" {
				continue
			}
			_ = os.Setenv(key, value)
		}
		return
	}
}

func dotEnvCandidates() []string {
	var paths []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		paths = append(paths,
			filepath.Join(dir, ".env"),
			filepath.Join(dir, "..", ".env"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths,
			filepath.Join(wd, ".env"),
			filepath.Join(wd, "..", ".env"),
		)
	}
	return paths
}

func openAIKeyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "meet-capture", "openai.key"), nil
}

func readOpenAIKey() (string, error) {
	loadDotEnv()
	if key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); key != "" {
		return key, nil
	}
	path, err := openAIKeyPath()
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("add an OpenAI API key first")
		}
		return "", err
	}
	key := strings.TrimSpace(string(raw))
	if key == "" {
		return "", errors.New("add an OpenAI API key first")
	}
	return key, nil
}

func saveOpenAIKey(key string) error {
	path, err := openAIKeyPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(path, []byte(key+"\n"), 0o600)
}

func openAIKeySaved() bool {
	_, err := readOpenAIKey()
	return err == nil
}

func answerQuestion(question, context, transcriptPath string) (openAIResult, error) {
	question = strings.TrimSpace(question)
	if question == "" || !strings.Contains(question, "?") {
		return openAIResult{}, errors.New("wait for a full question")
	}
	if len(question) > 2000 {
		question = question[:2000]
	}
	context = strings.TrimSpace(context)
	if len(context) > 24000 {
		context = context[len(context)-24000:]
	}

	key, err := readOpenAIKey()
	if err != nil {
		return openAIResult{}, err
	}
	answer, err := askOpenAIOne(key, question, context)
	if err != nil {
		return openAIResult{}, err
	}

	summaryPath := ""
	if strings.TrimSpace(transcriptPath) != "" {
		if err := allowedTranscript(transcriptPath); err != nil {
			return openAIResult{}, err
		}
		summaryPath = strings.TrimSuffix(transcriptPath, ".txt") + ".summary.txt"
		entry := "Q: " + question + "\nA: " + answer + "\n\n"
		file, err := os.OpenFile(summaryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return openAIResult{}, err
		}
		_, writeErr := file.WriteString(entry)
		closeErr := file.Close()
		if writeErr != nil {
			return openAIResult{}, writeErr
		}
		if closeErr != nil {
			return openAIResult{}, closeErr
		}
	}
	return openAIResult{Text: answer, SummaryPath: summaryPath}, nil
}

func summarizeTranscript(path string, emit func(string)) (openAIResult, error) {
	if err := allowedTranscript(path); err != nil {
		return openAIResult{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return openAIResult{}, err
	}
	transcript := strings.TrimSpace(string(raw))
	if transcript == "" {
		return openAIResult{}, errors.New("the transcript is empty")
	}
	transcript = clipRunes(transcript, 24000)

	key, err := readOpenAIKey()
	if err != nil {
		return openAIResult{}, err
	}
	answer, err := askOpenAI(key, transcript, emit)
	if err != nil {
		return openAIResult{}, err
	}

	summaryPath := strings.TrimSuffix(path, ".txt") + ".summary.txt"
	if err := os.WriteFile(summaryPath, []byte(answer+"\n"), 0o644); err != nil {
		return openAIResult{}, err
	}
	return openAIResult{Text: answer, SummaryPath: summaryPath}, nil
}

func clipRunes(text string, max int) string {
	if len(text) <= max {
		return text
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut]
}

func allowedTranscript(path string) error {
	// Normalize Windows paths to forward slashes before checking the
	// Downloads directory. The Linux implementation used "/Downloads/",
	// but Windows paths normally contain "\\Downloads\\".
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	normalizedPath := filepath.ToSlash(cleanPath)
	name := filepath.Base(cleanPath)

	if !filepath.IsAbs(cleanPath) ||
		!strings.Contains(normalizedPath, "/Downloads/") ||
		!strings.HasPrefix(name, "meet-") ||
		!strings.HasSuffix(name, ".txt") ||
		strings.HasSuffix(name, ".summary.txt") {
		return errors.New("choose a saved meeting transcript")
	}

	return nil
}

func askOpenAIOne(key, question, context string) (string, error) {
	user := "Question:\n" + question
	if context != "" {
		user += "\n\nTranscript so far:\n" + context
	}
	return completeOpenAI(key, "Answer this one question from a meeting or lesson. Use the transcript when it contains the answer. If it does not, answer the question yourself in clear language. Reply in plain text with the answer only.", user, nil)
}

func askOpenAI(key, transcript string, emit func(string)) (string, error) {
	return completeOpenAI(
		key,
		"The transcript is automatic speech recognition from a lesson or interview. Question marks are often missing and sentences may be incomplete. Identify the question the speaker is asking or introducing, and rewrite it as one clear question on a line starting with Q:. Then answer that question on the following lines starting with A:, using your own knowledge. Do not reply that there was no question.",
		"Transcript:\n\n"+transcript,
		emit,
	)
}

func completeOpenAI(key, system, user string, emit func(string)) (string, error) {
	request := map[string]any{
		"model":      openAIModel,
		"max_tokens": 320,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	}
	if emit != nil {
		request["stream"] = true
	}
	body, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if emit != nil {
		if resp.StatusCode != http.StatusOK {
			detail, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("OpenAI returned %s: %s", resp.Status, compactAPIError(detail))
		}
		return readChatStream(resp.Body, emit)
	}
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI returned %s: %s", resp.Status, compactAPIError(payload))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("OpenAI returned no answer")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func readChatStream(r io.Reader, emit func(string)) (string, error) {
	var built strings.Builder
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal([]byte(data), &chunk) != nil || len(chunk.Choices) == 0 {
			continue
		}
		piece := chunk.Choices[0].Delta.Content
		if piece == "" {
			continue
		}
		built.WriteString(piece)
		emit(piece)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	text := strings.TrimSpace(built.String())
	if text == "" {
		return "", errors.New("OpenAI returned no answer")
	}
	return text, nil
}

func compactAPIError(payload []byte) string {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(payload, &parsed) == nil && parsed.Error.Message != "" {
		return parsed.Error.Message
	}
	text := strings.TrimSpace(string(payload))
	if len(text) > 300 {
		return text[:300]
	}
	return text
}
