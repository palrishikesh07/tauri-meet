import { useEffect, useRef, useState } from "react";
import { listen } from "@tauri-apps/api/event";
import { getCurrentWindow } from "@tauri-apps/api/window";
import {
  listDevices,
  recordingStatus,
  startRecording,
  stopRecording,
  summarizeTranscript,
  type Device,
} from "./api";
import "./App.css";

// Window control functions
async function closeApp() {
  const active = getCurrentWindow();
  // Stop any recording if active
  try {
    await stopRecording().catch(() => null);
  } catch (err) {
    console.error("Error stopping recording:", err);
  }
  // Close the window
  await active.close();
}

// Utility Functions
function hasQuestion(text: string): boolean {
  return /\?|\b(how|what|why|when|where|who|which)\b|question/i.test(text);
}

function formatClock(totalSeconds: number): string {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

function messageOf(err: unknown): string {
  if (typeof err === "string") return err;
  if (err instanceof Error) return err.message;
  return "Something went wrong";
}

function App() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [source, setSource] = useState("");
  const [recording, setRecording] = useState(false);
  const [elapsed, setElapsed] = useState(0);
  const [activePath, setActivePath] = useState("");
  const [textPath, setTextPath] = useState("");
  const [transcript, setTranscript] = useState("");
  const [partial, setPartial] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [answer, setAnswer] = useState("");
  const [answering, setAnswering] = useState(false);
  const recordingRef = useRef(false);
  const stoppingRef = useRef(false);
  const summaryTask = useRef<Promise<unknown> | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [listed, status] = await Promise.all([
          listDevices(),
          recordingStatus(),
        ]);
        if (cancelled) return;
        setDevices(listed.devices);
        setSource(listed.defaultSource);
        setRecording(status.recording);
        setElapsed(status.elapsedSecs);
        setActivePath(status.path ?? "");
        setTextPath(status.textPath ?? "");
        setTranscript(status.transcript);
        setPartial(status.partial);
      } catch (err) {
        if (!cancelled) setError(messageOf(err));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!recording) return;
    const timer = window.setInterval(async () => {
      if (stoppingRef.current) return;
      try {
        const status = await recordingStatus();
        if (stoppingRef.current) return;
        if (!status.recording) {
          setRecording(false);
          return;
        }
        setRecording(true);
        setElapsed(status.elapsedSecs);
        setActivePath(status.path ?? "");
        setTextPath(status.textPath ?? "");
        setTranscript(status.transcript);
        setPartial(status.partial);
      } catch (err) {
        setError(messageOf(err));
      }
    }, 500);
    return () => window.clearInterval(timer);
  }, [recording]);

  useEffect(() => {
    recordingRef.current = recording;
  }, [recording]);

  const summaryGen = useRef(0);

  function requestSummary(path: string, force = false) {
    if (!path) return Promise.resolve();
    if (!force && summaryTask.current) return summaryTask.current;
    const gen = ++summaryGen.current;
    setAnswering(true);
    setError("");
    setAnswer("");
    const task = summarizeTranscript(path, gen)
      .then((result) => {
        if (gen !== summaryGen.current) return;
        setAnswer(result.text);
      })
      .catch((err) => {
        if (gen !== summaryGen.current) return;
        summaryTask.current = null;
        setError(messageOf(err));
      })
      .finally(() => {
        if (gen === summaryGen.current) setAnswering(false);
      });
    summaryTask.current = task;
    return task;
  }

  useEffect(() => {
    const unlisten = listen<{ ticket: number; text: string }>("answer-delta", (event) => {
      if (event.payload.ticket !== summaryGen.current) return;
      setAnswer((current) => current + event.payload.text);
    });
    return () => {
      void unlisten.then((stop) => stop());
    };
  }, []);

  useEffect(() => {
    if (!recording || !textPath || summaryTask.current || !hasQuestion(transcript)) return;
    void requestSummary(textPath);
  }, [recording, textPath, transcript]);

  useEffect(() => {
    const unlistenClose = getCurrentWindow().onCloseRequested(async (event) => {
      if (!recordingRef.current) return;
      event.preventDefault();
      recordingRef.current = false;
      await getCurrentWindow().destroy();
    });
    
    // Keyboard shortcuts for closing
    const handleKeyDown = (e: KeyboardEvent) => {
      // Ctrl+Q (Windows/Linux) or Cmd+Q (macOS)
      if ((e.ctrlKey || e.metaKey) && e.key === "q") {
        e.preventDefault();
        void closeApp();
      }
      // Alt+F4 is handled by the OS/Tauri
    };
    
    window.addEventListener("keydown", handleKeyDown);
    
    return () => {
      void unlistenClose.then((stop) => stop());
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, []);

  async function toggle() {
    setBusy(true);
    setError("");
    try {
      if (recording) {
        recordingRef.current = false;
        stoppingRef.current = true;
        setAnswering(true);
        const saved = await stopRecording();
        setRecording(false);
        setElapsed(0);
        setPartial("");
        setActivePath(saved.path);
        setTextPath(saved.textPath);
        const spoken = saved.transcript.trim();
        if (spoken) setTranscript(spoken);
        if (!saved.textPath || !spoken) {
          setError("The transcript was empty, so there was nothing to answer.");
        } else if (summaryTask.current) {
          await summaryTask.current;
        } else {
          await requestSummary(saved.textPath);
        }
      } else {
        const started = await startRecording(source);
        recordingRef.current = true;
        summaryTask.current = null;
        setAnswer("");
        setRecording(true);
        setElapsed(0);
        setTranscript("");
        setPartial("");
        setTextPath(started.textPath);
        setActivePath(started.path);
      }
    } catch (err) {
      recordingRef.current = false;
      setError(messageOf(err));
      const status = await recordingStatus().catch(() => null);
      if (status) {
        setRecording(status.recording);
        setElapsed(status.elapsedSecs);
        setActivePath(status.path ?? "");
      }
    } finally {
      stoppingRef.current = false;
      setAnswering(false);
      setBusy(false);
    }
  }

  const selected = devices.find((device) => device.id === source);
  const outputLabel =
    devices.length === 0
      ? "No playback devices found"
      : selected
        ? `${selected.label}${selected.isDefault ? " (current)" : ""}`
        : "Choose a playback output";

  return (
    <main className="app">
      <div style={{ position: "absolute", top: 10, right: 10 }}>
        <button
          type="button"
          onClick={closeApp}
          style={{
            background: "#ef4444",
            color: "white",
            border: "none",
            padding: "8px 12px",
            borderRadius: "4px",
            cursor: "pointer",
            fontSize: "12px",
            fontWeight: "500",
          }}
          title="Close application"
        >
          ✕ Quit
        </button>
      </div>
      <section className="panel">
        <h2>Answer</h2>
        <div className="transcript answers">
          {answer ? (
            <p>{answer}</p>
          ) : (
            <p className="hint">
              {answering
                ? "Answering the question…"
                : "The answer shows up here once a question is heard."}
            </p>
          )}
        </div>
      </section>

      <section className="panel">
        <label id="output-label">Playback output</label>
        <OutputPicker
          label={outputLabel}
          devices={devices}
          source={source}
          disabled={recording || devices.length === 0}
          onChange={setSource}
        />
        <p className="hint">
          {selected
            ? "This is the speaker or headset the meeting is playing through. Pick the one you are listening on."
            : "Connect speakers or a headset, then reopen the app."}
        </p>

        <div className="record-row">
          <button
            type="button"
            className={recording ? "record stop" : "record"}
            onClick={toggle}
            disabled={busy || (!recording && devices.length === 0)}
            aria-pressed={recording}
          >
            <span className="dot" />
            {recording ? "Stop" : "Record"}
          </button>
          <div>
            <p className="clock">{formatClock(elapsed)}</p>
            <p className="hint">
              {recording
                ? "Saving the audio and a .txt transcript. A line appears after a short pause."
                : "Start while you can hear the other people. The first moment loads the speech model."}
            </p>
          </div>
        </div>
        {activePath ? <p className="path">{activePath}</p> : null}
        {error ? <p className="error">{error}</p> : null}
      </section>

      <section className="panel">
        <h2>Transcript</h2>
        <div className="transcript">
          {transcript || partial ? (
            <>
              {transcript ? <p>{transcript}</p> : null}
              {partial ? <p className="partial">{partial}</p> : null}
            </>
          ) : (
            <p className="hint">
              {recording
                ? "Listening for speech…"
                : "Lines appear here a few seconds after someone speaks, and the same text is saved next to the audio."}
            </p>
          )}
        </div>
      </section>

      <p className="fine">
        Record only conversations you are allowed to record. Files stay on this
        computer.
      </p>
    </main>
  );
}

interface OutputPickerProps {
  label: string;
  devices: Device[];
  source: string;
  disabled: boolean;
  onChange: (id: string) => void;
}

function OutputPicker({
  label,
  devices,
  source,
  disabled,
  onChange,
}: OutputPickerProps) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const close = (event: MouseEvent) => {
      if (!root.current?.contains(event.target as Node)) setOpen(false);
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", close);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", close);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div className="picker" ref={root}>
      <button
        type="button"
        className="picker-button"
        aria-labelledby="output-label"
        aria-haspopup="listbox"
        aria-expanded={open}
        disabled={disabled}
        onClick={() => setOpen((value) => !value)}
      >
        <span>{label}</span>
        <span className="picker-chevron" aria-hidden="true" />
      </button>
      {open ? (
        <ul className="picker-menu" role="listbox" aria-labelledby="output-label">
          {devices.map((device) => {
            const text = `${device.label}${device.isDefault ? " (current)" : ""}`;
            const active = device.id === source;
            return (
              <li key={device.id} role="presentation">
                <button
                  type="button"
                  role="option"
                  aria-selected={active}
                  className={active ? "picker-option active" : "picker-option"}
                  onClick={() => {
                    onChange(device.id);
                    setOpen(false);
                  }}
                >
                  {text}
                </button>
              </li>
            );
          })}
        </ul>
      ) : null}
    </div>
  );
}

export default App;
