package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "devices":
		writeJSON(func() (any, error) {
			return listDevices()
		})
	case "list":
		writeJSON(func() (any, error) {
			return listRecordings()
		})
	case "key-status":
		writeJSON(func() (any, error) {
			return keyStatus{Saved: openAIKeySaved()}, nil
		})
	case "save-key":
		if err := saveOpenAIKey(os.Getenv("MEET_OPENAI_KEY")); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "answer":
		fs := flag.NewFlagSet("answer", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		text := fs.String("text", "", "one spoken question")
		context := fs.String("context", "", "transcript so far")
		file := fs.String("file", "", "optional meet-*.txt path; the answer is appended beside it")
		if err := fs.Parse(os.Args[2:]); err != nil {
			os.Exit(2)
		}
		writeJSON(func() (any, error) {
			return answerQuestion(*text, *context, *file)
		})
	case "summarize":
		fs := flag.NewFlagSet("summarize", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)

		file := fs.String(
			"file",
			"",
			"path to a meet-*.txt transcript",
		)

		topic := fs.String(
			"topic",
			"",
			"interview topic or technology",
		)

		stream := fs.Bool(
			"stream",
			false,
			"print the answer as it arrives",
		)

		if err := fs.Parse(os.Args[2:]); err != nil {
			os.Exit(2)
		}

		if *stream {
			out := json.NewEncoder(os.Stdout)

			result, err := summarizeTranscript(
				*file,
				*topic,
				func(delta string) {
					_ = out.Encode(
						map[string]string{
							"delta": delta,
						},
					)
				},
			)

			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			_ = out.Encode(map[string]any{
				"done":        true,
				"text":        result.Text,
				"summaryPath": result.SummaryPath,
			})

			return
		}

		writeJSON(func() (any, error) {
			return summarizeTranscript(
				*file,
				*topic,
				nil,
			)
		})
	case "record":
		fs := flag.NewFlagSet("record", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		source := fs.String("source", "", "playback monitor id (sink name + .monitor)")
		if err := fs.Parse(os.Args[2:]); err != nil {
			os.Exit(2)
		}
		if err := record(*source); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
}

func writeJSON(load func() (any, error)) {
	value, err := load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `meetrec records meeting audio from a playback monitor (what you hear), not the microphone.

  meetrec devices
  meetrec list
  meetrec record [--source MONITOR_ID]
  meetrec answer --text QUESTION [--context TRANSCRIPT] [--file TRANSCRIPT.txt]
  meetrec summarize --file TRANSCRIPT.txt [--topic TECHNOLOGY]`)
}
