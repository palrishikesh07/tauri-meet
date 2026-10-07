# Meet Capture

Desktop app that records the **other side of a meeting**: the audio playing through your speakers or headset. It does not open the microphone. Each take is saved as an `.ogg` file in your Downloads folder, with a `.txt` transcript beside it.

Speech to text uses a local Whisper model. A line appears after a short pause. Closing the window stops the recorder immediately instead of waiting on a background process.

After a transcript is saved, Summarize sends that text to OpenAI and writes a sibling `.summary.txt` with a short summary and answers to the questions in it. The API key is stored only in `~/.config/meet-capture/openai.key`.

## Why this stack

- **Tauri** is the window. Rust starts and stops the recorder and keeps the recording alive while the window is open.
- **Go** (`engine/`) is the recorder. It asks PipeWire which speakers are in use, then runs `ffmpeg` against that output’s **monitor**. A monitor is a copy of sound heading to the speakers, which is what you hear from the meeting.
- Your own voice is a microphone **source**. Those devices are never listed.

## Run

```bash
npm install
npm run tauri dev
```

The dev script builds `engine/meetrec` before the window opens. You need Go, Rust, Node, `ffmpeg`, and PipeWire (`pactl`).

Try the recorder on its own:

```bash
go -C engine test .
go -C engine build -o meetrec .
./engine/meetrec devices
./engine/meetrec record
# Ctrl+C writes the file and stops
./engine/meetrec list
```

Pick the playback device the meeting is actually using (laptop speakers, HDMI, or a Bluetooth headset) before you press Record.
