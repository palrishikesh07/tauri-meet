import { invoke } from "@tauri-apps/api/core";

export interface Device {
  id: string;
  label: string;
  detail: string;
  isDefault: boolean;
}

export interface DeviceList {
  downloadsDir: string;
  defaultSource: string;
  devices: Device[];
}

export interface RecordingStatus {
  recording: boolean;
  path: string | null;
  textPath: string | null;
  source: string | null;
  elapsedSecs: number;
  transcript: string;
  partial: string;
}

export interface ReadyMessage {
  status: string;
  path: string;
  source: string;
  textPath: string;
}

export interface SavedTake {
  path: string;
  bytes: number;
  textPath: string;
  transcript: string;
}

export interface OpenAIResult {
  text: string;
  summaryPath: string;
}

export async function listDevices(): Promise<DeviceList> {
  return invoke<DeviceList>("list_devices");
}

export async function recordingStatus(): Promise<RecordingStatus> {
  return invoke<RecordingStatus>("recording_status");
}

export async function startRecording(source?: string): Promise<ReadyMessage> {
  return invoke<ReadyMessage>("start_recording", { source: source || null });
}

export async function stopRecording(): Promise<SavedTake> {
  return invoke<SavedTake>("stop_recording");
}

export async function summarizeTranscript(
  path: string,
  ticket: number,
  topic = "",
): Promise<OpenAIResult> {
  return invoke<OpenAIResult>("summarize_transcript", {
    path,
    ticket,
    topic,
  });
}

export async function summarizeTranscriptLocal(
  path: string,
  ticket: number,
  topic = "",
): Promise<OpenAIResult> {
  return invoke<OpenAIResult>(
    "summarize_transcript_local",
    {
      path,
      ticket,
      topic,
    },
  );
}


export async function saveTranscript(
  path: string,
  text: string,
): Promise<void> {
  return invoke<void>("save_transcript", {
    path,
    text,
  });
}
