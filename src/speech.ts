import { saveTranscript } from "./api";

type SpeechHandlers = {
  onStatus: (text: string) => void;
  onTranscript: (text: string, partial: string) => void;
  onError: (text: string) => void;
};

const sampleRate = 16000;
const maxSamples = sampleRate * 5;
const minSamples = Math.floor(sampleRate * 0.7);
const silenceSamples = Math.floor(sampleRate * 0.55);

let worker: Worker | null = null;
let handlers: SpeechHandlers | null = null;
let ready = false;
let busy = false;
let textPath = "";
let lines: string[] = [];
let carry = new Float32Array(0);
let silent = 0;
let voiced = 0;
const queue: Float32Array[] = [];

export function bindSpeech(next: SpeechHandlers) {
  handlers = next;
  ensureWorker();
}

export function setTranscriptTarget(path: string) {
  textPath = path;
  lines = [];
  carry = new Float32Array(0);
  silent = 0;
  voiced = 0;
  queue.length = 0;
}

export function pushPcm(encoded: string) {
  const chunk = decodePcm(encoded);
  carry = concat(carry, chunk);
  if (rms(chunk) >= 0.012) {
    voiced += chunk.length;
    silent = 0;
  } else if (voiced > 0) {
    silent += chunk.length;
  } else {
    carry = new Float32Array(0);
  }

  if (voiced >= maxSamples || (voiced >= minSamples && silent >= silenceSamples)) {
    const phrase = carry.slice(0, carry.length - (silent >= silenceSamples ? silent : 0));
    carry = new Float32Array(0);
    silent = 0;
    voiced = 0;
    if (phrase.length >= minSamples) queue.push(phrase);
  }
  pump();
}

export function flushSpeech() {
  if (voiced >= minSamples && carry.length > 0) {
    queue.push(carry);
  }
  carry = new Float32Array(0);
  silent = 0;
  voiced = 0;
  pump();
  return waitUntilIdle();
}

function ensureWorker() {
  if (worker) return;
  worker = new Worker(new URL("./transcribe.worker.ts", import.meta.url), {
    type: "module",
  });
  worker.onmessage = (event: MessageEvent) => {
    const message = event.data as {
      type: string;
      text?: string;
    };
    if (message.type === "progress" && message.text) {
      handlers?.onStatus(message.text);
      return;
    }
    if (message.type === "ready") {
      ready = true;
      handlers?.onStatus("Speech model ready");
      pump();
      return;
    }
    if (message.type === "error") {
      busy = false;
      handlers?.onError(message.text || "Speech model failed");
      pump();
      return;
    }
    if (message.type === "result") {
      busy = false;
      const text = clean(message.text || "");
      if (text && !skip(text)) {
        lines.push(text);
        const full = lines.join("\n");
        handlers?.onTranscript(full, "");
        if (textPath) {
          void saveTranscript(textPath, full + "\n").catch((err) => {
            handlers?.onError(err instanceof Error ? err.message : String(err));
          });
        }
      } else {
        handlers?.onTranscript(lines.join("\n"), "");
      }
      pump();
    }
  };
  worker.postMessage({ type: "load" });
  handlers?.onStatus("Loading the speech model…");
}

function pump() {
  if (!worker || !ready || busy || queue.length === 0) return;
  busy = true;
  const audio = queue.shift();
  if (!audio) {
    busy = false;
    return;
  }
  handlers?.onTranscript(lines.join("\n"), "Transcribing…");
  worker.postMessage({ type: "transcribe", audio }, [audio.buffer]);
}

function waitUntilIdle() {
  return new Promise<void>((resolve) => {
    const started = Date.now();
    const timer = window.setInterval(() => {
      if ((!busy && queue.length === 0) || Date.now() - started > 2000) {
        window.clearInterval(timer);
        resolve();
      }
    }, 50);
  });
}

function decodePcm(encoded: string) {
  const binary = atob(encoded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i += 1) bytes[i] = binary.charCodeAt(i);
  const view = new DataView(bytes.buffer);
  const samples = new Float32Array(Math.floor(bytes.length / 2));
  for (let i = 0; i < samples.length; i += 1) {
    samples[i] = view.getInt16(i * 2, true) / 32768;
  }
  return samples;
}

function concat(left: Float32Array, right: Float32Array) {
  const merged = new Float32Array(left.length + right.length);
  merged.set(left);
  merged.set(right, left.length);
  return merged;
}

function rms(samples: Float32Array) {
  if (samples.length === 0) return 0;
  let sum = 0;
  for (let i = 0; i < samples.length; i += 1) sum += samples[i] * samples[i];
  return Math.sqrt(sum / samples.length);
}

function clean(text: string) {
  return text.replace(/\s+/g, " ").trim();
}

function skip(text: string) {
  const folded = text.toLowerCase().replace(/[.!?]/g, "").trim();
  return (
    folded === "" ||
    folded === "you" ||
    folded === "thank you" ||
    folded === "thanks for watching" ||
    folded === "thanks for watching."
  );
}
