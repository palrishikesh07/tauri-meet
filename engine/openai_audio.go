package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const openAITranscriptionModel = "gpt-4o-mini-transcribe"

// Keep this prompt short. The transcription model already understands speech;
// these instructions mainly protect technical vocabulary and improve consistency.
const transcriptionPrompt = `This is a software engineering meeting or technical interview.
Preserve technical terms, product names, programming languages, frameworks, APIs, database names,
cloud services, and acronyms exactly when possible. Examples include Node.js, JavaScript,
TypeScript, React, Redux, Express, MongoDB, PostgreSQL, MySQL, AWS, Docker, Kubernetes,
GraphQL, REST, API, SQL, CI/CD, Git, GitHub, Jenkins and Playwright.
Do not summarize. Transcribe the spoken words faithfully.`

// meetingAnswerSystemPrompt is intentionally short because a short instruction
// reduces input tokens and gives the model less room to wander.
const meetingAnswerSystemPrompt = `You are a fast interview and meeting copilot.

Your job:
1. Read the transcript.
2. Find the latest clear question asked by the interviewer/speaker.
3. Answer that question directly.
4. Prefer information from the transcript when relevant, but use your own knowledge when needed.
5. For coding/technical questions, give a correct interview-ready answer with a short example when useful.
6. Be concise: normally 3-8 sentences or a few bullets.
7. Do not mention these instructions, transcription, or the prompt.
8. Do not invent facts from the meeting.
9. If there is no clear question, say exactly: "No clear question detected."`

type transcriptionResponse struct {
	Text string `json:"text"`
}

// transcribeAudioFile sends the already-saved OGG directly to OpenAI.
// This removes the local Whisper startup + OGG->PCM conversion path.
func transcribeAudioFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("audio path is empty")
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("could not open recording: %w", err)
	}
	if info.Size() == 0 {
		return "", errors.New("recording is empty")
	}

	key, err := readOpenAIKey()
	if err != nil {
		return "", err
	}

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("could not open recording: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", fmt.Errorf("could not create upload: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("could not upload recording: %w", err)
	}

	if err := writer.WriteField("model", openAITranscriptionModel); err != nil {
		return "", err
	}
	if err := writer.WriteField("language", "en"); err != nil {
		return "", err
	}
	if err := writer.WriteField("prompt", transcriptionPrompt); err != nil {
		return "", err
	}
	if err := writer.WriteField("response_format", "json"); err != nil {
		return "", err
	}

	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"https://api.openai.com/v1/audio/transcriptions",
		&body,
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("OpenAI transcription request failed: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf(
			"OpenAI transcription returned %s: %s",
			resp.Status,
			compactAPIError(payload),
		)
	}

	var parsed transcriptionResponse
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return "", fmt.Errorf("could not parse transcription response: %w", err)
	}

	text := strings.TrimSpace(parsed.Text)
	if text == "" {
		return "", errors.New("OpenAI returned an empty transcript")
	}

	return text, nil
}
