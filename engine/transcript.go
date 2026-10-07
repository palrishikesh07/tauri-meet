package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"os"
	"strings"
	"sync"
)

type textMessage struct {
	Status  string `json:"status"`
	Text    string `json:"text"`
	Partial string `json:"partial"`
}

type pcmMessage struct {
	Status string `json:"status"`
	Pcm    string `json:"pcm"`
}

// stdoutPublisher writes transcript updates after the recording-ready line.
type stdoutPublisher struct {
	mu sync.Mutex
	on bool
}

func (p *stdoutPublisher) enable() {
	p.mu.Lock()
	p.on = true
	p.mu.Unlock()
}

func (p *stdoutPublisher) publishPCM(pcm []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.on || len(pcm) == 0 {
		return
	}
	_ = json.NewEncoder(os.Stdout).Encode(pcmMessage{
		Status: "pcm",
		Pcm:    base64.StdEncoding.EncodeToString(pcm),
	})
}

func (p *stdoutPublisher) publish(text, partial string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.on {
		return
	}
	_ = json.NewEncoder(os.Stdout).Encode(textMessage{
		Status:  "text",
		Text:    text,
		Partial: partial,
	})
}

func writeReady(msg readyMessage) error {
	return json.NewEncoder(os.Stdout).Encode(msg)
}

// transcribePCM reads 16 kHz mono s16le audio until EOF.
// Audio is sent to Whisper every few seconds so text shows up during the recording,
// and again at the end so the last words are not dropped.
func transcribePCM(pcm io.Reader, textFile *os.File, pub *stdoutPublisher, client *whisperClient) error {
	jobs := make(chan []byte, 8)
	done := make(chan struct{})
	go func() {
		defer close(done)
		var lines []string
		for phrase := range jobs {
			pub.publish(strings.Join(lines, "\n"), "Transcribing…")
			raw, err := client.infer(phrase)
			if err != nil {
				pub.publish(strings.Join(lines, "\n"), err.Error())
				continue
			}
			text := cleanHyp(raw)
			if skipHallucination(text) || tooRepetitive(text) {
				pub.publish(strings.Join(lines, "\n"), "")
				continue
			}
			lines = append(lines, text)
			_, _ = textFile.WriteString(text + "\n")
			_ = textFile.Sync()
			pub.publish(strings.Join(lines, "\n"), "")
		}
	}()

	var phrase []byte
	silenceSamples := 0
	speechSamples := 0
	buf := make([]byte, 3200) // 100 ms
	flush := func(force bool) {
		if len(phrase) < 2 {
			return
		}
		if !force && speechSamples < 16000/2 {
			phrase = nil
			silenceSamples = 0
			speechSamples = 0
			return
		}
		if !force && !pcmLoud(phrase) {
			phrase = nil
			silenceSamples = 0
			speechSamples = 0
			return
		}
		jobs <- append([]byte(nil), phrase...)
		phrase = nil
		silenceSamples = 0
		speechSamples = 0
	}

	for {
		n, err := pcm.Read(buf)
		if n > 1 {
			chunk := append([]byte(nil), buf[:n-n%2]...)
			samples := len(chunk) / 2
			if pcmLoud(chunk) {
				phrase = append(phrase, chunk...)
				speechSamples += samples
				silenceSamples = 0
			} else if speechSamples > 0 {
				phrase = append(phrase, chunk...)
				silenceSamples += samples
			}
			// Keep phrases short so Whisper finishes while the meeting is still going.
			if speechSamples >= 16000*5 || (speechSamples > 0 && silenceSamples >= 16000*6/10) {
				flush(false)
			}
		}
		if err != nil {
			flush(true)
			close(jobs)
			<-done
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func tooRepetitive(text string) bool {
	fields := strings.Fields(strings.ToLower(text))
	if len(fields) < 8 {
		return false
	}
	counts := map[string]int{}
	for _, word := range fields {
		counts[word]++
		if counts[word] >= 6 {
			return true
		}
	}
	return false
}

func pcmLoud(pcm []byte) bool {
	if len(pcm) < 2 {
		return false
	}
	var sum float64
	n := 0
	for i := 0; i+1 < len(pcm); i += 2 {
		sample := int16(pcm[i]) | int16(pcm[i+1])<<8
		sum += float64(sample) * float64(sample)
		n++
	}
	return math.Sqrt(sum/float64(n)) >= 180
}

func cleanHyp(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, " \"'")
	return strings.Join(strings.Fields(text), " ")
}

func skipHallucination(text string) bool {
	switch strings.ToLower(text) {
	case "", "you", "thank you", "thanks", "thanks for watching",
		"thank you for watching", "subscribe", "music", "blank audio":
		return true
	default:
		return false
	}
}
