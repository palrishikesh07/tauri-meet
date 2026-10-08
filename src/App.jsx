import { useEffect, useRef, useState } from "react";
import { listen } from "@tauri-apps/api/event";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { invoke } from "@tauri-apps/api/core";

import {
  listDevices,
  recordingStatus,
  startRecording,
  stopRecording,
  summarizeTranscript,
  summarizeTranscriptLocal,
  saveTranscript,
} from "./api";

import "./App.css";

function hasQuestion(text) {
  return /\?|\b(how|what|why|when|where|who|which)\b|question/i.test(text);
}

function formatClock(totalSeconds) {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

function messageOf(err) {
  if (typeof err === "string") return err;
  if (err instanceof Error) return err.message;
  return "Something went wrong";
}

function App() {
  const [devices, setDevices] = useState([]);
  const [source, setSource] = useState("");
  const [downloadsDir, setDownloadsDir] = useState("");
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
  const [cancelling, setCancelling] = useState(false);

  const [aiProvider, setAiProvider] = useState("openai");

  // Interview topic / technology used to focus the OpenAI answer.
  const [topic, setTopic] = useState("");

  const recordingRef = useRef(false);
  const stoppingRef = useRef(false);
  const summaryTask = useRef(null);
  const summaryGen = useRef(0);

  const appWindow = getCurrentWindow();

  // Keep this Tauri window visible locally but protected from supported
  // screen-capture APIs such as Windows WDA_EXCLUDEFROMCAPTURE.
  useEffect(() => {
    const enableContentProtection = async () => {
      try {
        await appWindow.setContentProtected(true);
        console.log("Screen capture protection enabled");
      } catch (err) {
        console.error("Failed to enable content protection:", err);
      }
    };

    void enableContentProtection();
  }, [appWindow]);

  // Load playback devices and current recording state.
  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        const [listed, status] = await Promise.all([
          listDevices(),
          recordingStatus(),
        ]);

        if (cancelled) return;

        setDevices(listed.devices || []);
        setDownloadsDir(listed.downloadsDir || "");
        setSource(listed.defaultSource || "");
        setRecording(status.recording);
        setElapsed(status.elapsedSecs || 0);
        setActivePath(status.path ?? "");
        setTextPath(status.textPath ?? "");
        setTranscript(status.transcript || "");
        setPartial(status.partial || "");
      } catch (err) {
        if (!cancelled) setError(messageOf(err));
      }
    })();

    return () => {
      cancelled = true;
    };
  }, []);

  // Poll recording state while recording.
  useEffect(() => {
    if (!recording) return undefined;

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
        setElapsed(status.elapsedSecs || 0);
        setActivePath(status.path ?? "");
        setTextPath(status.textPath ?? "");
        setTranscript(status.transcript || "");
        setPartial(status.partial || "");
      } catch (err) {
        setError(messageOf(err));
      }
    }, 500);

    return () => window.clearInterval(timer);
  }, [recording]);

  useEffect(() => {
    recordingRef.current = recording;
  }, [recording]);

  // Ask OpenAI about the latest transcript, optionally focused on the
  // selected interview topic / technology.
  function requestSummary(
    path,
    force = false,
  ) {
    if (!path) {
      return Promise.resolve();
    }

    if (
      !force &&
      summaryTask.current
    ) {
      return summaryTask.current;
    }

    const gen =
      ++summaryGen.current;

    setAnswering(true);
    setError("");
    setAnswer("");

    const request =
      aiProvider === "local"
        ? summarizeTranscriptLocal(
          path,
          gen,
          topic,
        )
        : summarizeTranscript(
          path,
          gen,
          topic,
        );

    const task = request
      .then((result) => {
        if (
          gen !==
          summaryGen.current
        ) {
          return;
        }

        setAnswer(result.text);
      })
      .catch((err) => {
        if (
          gen !==
          summaryGen.current
        ) {
          return;
        }

        summaryTask.current = null;

        setError(
          messageOf(err),
        );
      })
      .finally(() => {
        if (
          gen ===
          summaryGen.current
        ) {
          setAnswering(false);
        }
      });

    summaryTask.current = task;

    return task;
  }

  // Stream answer deltas from the Rust/OpenAI pipeline.
  useEffect(() => {
    let mounted = true;

    const unlisten = listen("answer-delta", (event) => {
      if (!mounted) return;
      if (event.payload.ticket !== summaryGen.current) return;

      setAnswer((current) => current + event.payload.text);
    });

    return () => {
      mounted = false;
      void unlisten.then((stop) => stop());
    };
  }, []);

  // If a question becomes visible during recording, start an answer.
  useEffect(() => {
    if (!recording || !textPath || summaryTask.current || !hasQuestion(transcript)) {
      return;
    }

    void requestSummary(textPath);
  }, [recording, textPath, transcript, topic]);

  async function cancelAnswer() {
    if (cancelling) return;

    // Invalidate the current UI request first so no late stream chunk
    // from the old request can appear in the next answer.
    ++summaryGen.current;
    summaryTask.current = null;

    setCancelling(true);
    setAnswer("");
    setAnswering(false);
    setError("");

    try {
      await invoke("cancel_ai_request");
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setCancelling(false);
      setBusy(false);
    }
  }

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

        const spoken = (saved.transcript || "").trim();

        if (spoken) {
          setTranscript(spoken);
        }

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
        setTextPath(started.textPath || "");
        setActivePath(started.path || "");
      }
    } catch (err) {
      recordingRef.current = false;
      setError(messageOf(err));

      const status = await recordingStatus().catch(() => null);

      if (status) {
        setRecording(status.recording);
        setElapsed(status.elapsedSecs || 0);
        setActivePath(status.path ?? "");
        setTextPath(status.textPath ?? "");
      }
    } finally {
      stoppingRef.current = false;
      setAnswering(false);
      setBusy(false);
    }
  }

  const handleRecordWheel = (event) => {
    event.preventDefault();
    event.stopPropagation();

    const answerSection = document.querySelector(".answers");
    if (!answerSection) return;

    answerSection.scrollTop += event.deltaY;
  };

  const selected = devices.find((device) => device.id === source);

  const outputLabel =
    devices.length === 0
      ? "No playback devices found"
      : selected
        ? `${selected.label}${selected.isDefault ? " (current)" : ""}`
        : "Choose a playback output";

  return (
    <main className="app">
      <section className="panel">
        <div className="transcript answers">
          {answer ? (
            <div className="answer-content">
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                components={{
                  code({ className, children, ...props }) {
                    const match = /language-(\w+)/.exec(className || "");

                    return (
                      <pre className="code-block">
                        <code
                          className={match ? `language-${match[1]}` : ""}
                          {...props}
                        >
                          {String(children).replace(/\n$/, "")}
                        </code>
                      </pre>
                    );
                  },
                }}
              >
                {answer}
              </ReactMarkdown>
            </div>
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
        <h2>Transcript</h2>

        <textarea
          className="transcript-editor"
          value={transcript}
          onChange={(event) => setTranscript(event.target.value)}
          placeholder={
            recording
              ? "Listening for speech…"
              : "Transcript appears here. You can edit it or type your own question, for example: Explain loops in Node.js"
          }
          spellCheck={false}
          aria-label="Transcript or custom question"
        />

        {partial ? (
          <p className="partial transcript-partial">{partial}</p>
        ) : null}

        <div className="transcript-actions">
          <button
            type="button"
            className="ask-transcript"
            onClick={async () => {
              const text = transcript.trim();

              if (!text) {
                setError("Type a question in the Transcript box first.");
                return;
              }

              try {
                setBusy(true);
                setError("");

                // A manually typed question does not need a previous recording.
                // Create a fresh transcript file in Downloads so both OpenAI
                // and local Qwen use exactly the text entered in the textarea.
                const manualPath =
                  textPath ||
                  `${downloadsDir}/meet-manual-question-${Date.now()}.txt`;

                await saveTranscript(manualPath, text);
                setTextPath(manualPath);
                await requestSummary(manualPath, true);
              } catch (err) {
                setError(messageOf(err));
              } finally {
                setBusy(false);
              }
            }}
          >
            {answering ? "Answering…" : "Ask AI"}
          </button>

          <span className="hint">
            Edit the transcript or type your own question, then click Ask AI.
          </span>
        </div>

        {textPath ? <p className="path">{textPath}</p> : null}
      </section>
      <section className="panel topic-panel">
        <label htmlFor="interview-topic">Interview Topic / Technology</label>

        <input
          id="interview-topic"
          type="text"
          value={topic}
          onChange={(event) => setTopic(event.target.value)}
          placeholder="e.g. Node.js, React, AWS, MongoDB"
          disabled={recording || busy}
          autoComplete="off"
        />

        <p className="hint">
          Optional. This topic is sent with the transcript so the answer stays
          focused on the selected technology.
        </p>
      </section>

      <section className="panel">
        <label htmlFor="ai-provider">
          AI Provider
        </label>

        <select
          id="ai-provider"
          value={aiProvider}
          onChange={(event) =>
            setAiProvider(
              event.target.value,
            )
          }
          disabled={
            recording || busy
          }
        >
          <option value="openai">
            OpenAI — Cloud
          </option>

          <option value="local">
            Qwen3 4B — Local
          </option>
        </select>

        <p className="hint">
          {aiProvider === "local"
            ? "Runs locally through Ollama. Internet is not required for answer generation."
            : "Uses your existing OpenAI configuration."}
        </p>
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
          <div className="record-control" onWheel={handleRecordWheel}>
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

            {answering ? (
              <button
                type="button"
                className="cancel-answer"
                onClick={() => void cancelAnswer()}
                disabled={cancelling}
                style={{
                  marginTop: "8px",
                  minWidth: "110px",
                  height: "34px",
                  padding: "0 14px",
                  borderRadius: "18px",
                  border: "1px solid rgba(255, 100, 100, 0.45)",
                  background: cancelling
                    ? "rgba(255, 70, 70, 0.08)"
                    : "rgba(255, 70, 70, 0.16)",
                  color: "#fff",
                  cursor: cancelling ? "wait" : "pointer",
                  opacity: cancelling ? 0.65 : 1,
                  fontSize: "13px",
                  fontWeight: 600,
                }}
              >
                {cancelling ? "Cancelling…" : "⛔ Cancel / Reset"}
              </button>
            ) : null}
          </div>

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


    </main>
  );
}

function OutputPicker({
  label,
  devices,
  source,
  disabled,
  onChange,
}) {
  const [open, setOpen] = useState(false);
  const root = useRef(null);

  useEffect(() => {
    if (!open) return undefined;

    const close = (event) => {
      if (!root.current?.contains(event.target)) {
        setOpen(false);
      }
    };

    const onKey = (event) => {
      if (event.key === "Escape") {
        setOpen(false);
      }
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
        <ul
          className="picker-menu"
          role="listbox"
          aria-labelledby="output-label"
        >
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
