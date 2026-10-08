package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	ollamaURL   = "http://localhost:11434/api/chat"
	ollamaModel = "qwen3:4b-instruct"
)

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model     string          `json:"model"`
	Messages  []ollamaMessage `json:"messages"`
	Stream    bool            `json:"stream"`
	KeepAlive string          `json:"keep_alive,omitempty"`
	Think     *bool           `json:"think,omitempty"`
	Options   ollamaOptions   `json:"options,omitempty"`
}
type ollamaOptions struct {
	NumCtx      int     `json:"num_ctx,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaResponse struct {
	Model   string `json:"model"`
	Message struct {
		Role     string `json:"role"`
		Content  string `json:"content"`
		Thinking string `json:"thinking,omitempty"`
	} `json:"message"`

	Done  bool   `json:"done"`
	Error string `json:"error,omitempty"`
}

func checkOllama() error {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(
		"http://localhost:11434/api/tags",
	)

	if err != nil {
		return errors.New(
			"Ollama is not running. Start Ollama and try again.",
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"Ollama returned HTTP %d",
			resp.StatusCode,
		)
	}

	return nil
}

func localSummarizeTranscript(
	path string,
	topic string,
	emit func(string),
) (openAIResult, error) {

	if err := checkOllama(); err != nil {
		return openAIResult{}, err
	}

	resolvedPath, err := resolveTranscriptPath(path)
	if err != nil {
		return openAIResult{}, err
	}

	raw, err := os.ReadFile(resolvedPath)
	if err != nil {
		return openAIResult{}, err
	}

	transcript := strings.TrimSpace(
		string(raw),
	)

	if transcript == "" {
		return openAIResult{},
			errors.New("the transcript is empty")
	}

	// IMPORTANT:
	// Your laptop has only 12 GB RAM.
	// Keep the transcript relatively small.
	transcript = clipTailRunes(
		transcript,
		1200,
	)

	topic = strings.TrimSpace(topic)

	systemPrompt := `
You are a senior software interview assistant.

Answer the interview question using only the context below.

Rules:
- Answer in 5-10 lines maximum.
- Be direct and technically accurate.
- Do not repeat the question.
- Use bullets when useful.
- No long explanation.
- If code is required, give a short example.
`

	userPrompt :=
		"Latest interview transcript:\n\n" +
			transcript

	if topic != "" {
		userPrompt =
			"Interview Topic / Technology:\n" +
				topic +
				"\n\n" +
				userPrompt
	}

	think := false

	requestBody := ollamaRequest{
		Model: ollamaModel,

		Messages: []ollamaMessage{
			{
				Role: "system",
				Content: strings.TrimSpace(
					systemPrompt,
				),
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},

		Stream:    true,
		KeepAlive: "45m",
		Think:     &think,

		Options: ollamaOptions{
			// Important for 12 GB RAM.
			NumCtx: 1536,

			// Low temperature gives more
			// deterministic interview answers.
			Temperature: 0.1,

			TopP: 0.8,

			// Keep generated answer reasonably short.
			NumPredict: 300,
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return openAIResult{}, err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		ollamaURL,
		bytes.NewReader(body),
	)

	if err != nil {
		return openAIResult{}, err
	}

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	client := &http.Client{
		Timeout: 180 * time.Second,
	}

	resp, err := client.Do(req)

	if err != nil {
		return openAIResult{},
			fmt.Errorf(
				"could not connect to Ollama: %w",
				err,
			)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return openAIResult{},
			fmt.Errorf(
				"Ollama returned HTTP %d",
				resp.StatusCode,
			)
	}

	scanner := bufio.NewScanner(
		resp.Body,
	)

	scanner.Buffer(
		make([]byte, 4096),
		1024*1024,
	)

	var answer strings.Builder

	for scanner.Scan() {

		line := strings.TrimSpace(
			scanner.Text(),
		)

		if line == "" {
			continue
		}

		var chunk ollamaResponse

		if err := json.Unmarshal(
			[]byte(line),
			&chunk,
		); err != nil {
			continue
		}

		if chunk.Error != "" {
			return openAIResult{},
				errors.New(chunk.Error)
		}

		delta := chunk.Message.Content

		if delta != "" {

			answer.WriteString(delta)

			if emit != nil {
				emit(delta)
			}
		}

		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return openAIResult{}, err
	}

	result := strings.TrimSpace(
		answer.String(),
	)

	if result == "" {
		return openAIResult{},
			errors.New(
				"Ollama returned an empty answer",
			)
	}

	summaryPath :=
		strings.TrimSuffix(
			resolvedPath,
			".txt",
		) + ".summary.txt"

	if err := os.WriteFile(
		summaryPath,
		[]byte(result+"\n"),
		0o644,
	); err != nil {
		return openAIResult{}, err
	}

	return openAIResult{
		Text:        result,
		SummaryPath: summaryPath,
	}, nil
}
