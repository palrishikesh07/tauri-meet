/// <reference lib="webworker" />
import { env, pipeline } from "@huggingface/transformers";

const scope = self as unknown as DedicatedWorkerGlobalScope;

env.allowLocalModels = false;

let transcriber: Awaited<
  ReturnType<typeof pipeline<"automatic-speech-recognition">>
> | null = null;

scope.onmessage = async (event: MessageEvent) => {
  const message = event.data as
    | { type: "load" }
    | { type: "transcribe"; audio: Float32Array };

  if (message.type === "load") {
    try {
      transcriber = await pipeline(
        "automatic-speech-recognition",
        "Xenova/whisper-base.en",
        {
          dtype: "q8",
          device: "wasm",
          progress_callback: (item) => {
            if (item.status === "progress") {
              scope.postMessage({
                type: "progress",
                text: `Downloading ${item.file} ${Math.round(item.progress)}%`,
              });
            } else if (item.status === "ready") {
              scope.postMessage({ type: "progress", text: "Speech model ready" });
            }
          },
        },
      );
      scope.postMessage({ type: "ready" });
    } catch (err) {
      scope.postMessage({ type: "error", text: messageOf(err) });
    }
    return;
  }

  if (!transcriber) {
    scope.postMessage({ type: "error", text: "Speech model is not ready yet" });
    return;
  }

  try {
    const result = await transcriber(message.audio, {
      language: "english",
      task: "transcribe",
    });
    const text = Array.isArray(result) ? result.map((item) => item.text).join(" ") : result.text;
    scope.postMessage({ type: "result", text: text.trim() });
  } catch (err) {
    scope.postMessage({ type: "error", text: messageOf(err) });
  }
};

function messageOf(err: unknown) {
  if (err instanceof Error) return err.message;
  return String(err);
}
