import { invoke } from "@tauri-apps/api/core";

export type Device = {
  id: string;
  label: string;
  detail: string;
  isDefault: boolean;
};

export type DeviceList = {
  downloadsDir: string;
  defaultSource: string;
  devices: Device[];
};

export type RecordingFile = {
  name: string;
  path: string;
  bytes: number;
  modified: string;
  textPath?: string;
  summaryPath?: string;
};

export type RecordingStatus = {
  recording: boolean;
  path: string | null;
  textPath: string | null;
  source: string | null;
  elapsedSecs: number;
  transcript: string;
  partial: string;
};

export type SavedTake = {
  path: string;
  bytes: number;
  textPath: string;
  transcript: string;
};

export const listDevices = () => invoke<DeviceList>("list_devices");

export const listRecordings = () => invoke<RecordingFile[]>("list_recordings");

export const recordingStatus = () => invoke<RecordingStatus>("recording_status");

export const startRecording = (source: string) =>
  invoke<{ status: string; path: string; source: string; textPath: string }>(
    "start_recording",
    { source },
  );

export const saveTranscript = (path: string, text: string) =>
  invoke<void>("save_transcript", { path, text });

export const stopRecording = () => invoke<SavedTake>("stop_recording");

export const openaiKeyStatus = () => invoke<{ saved: boolean }>("openai_key_status");

export const saveOpenAIKey = (key: string) =>
  invoke<void>("save_openai_key", { key });

export const summarizeTranscript = (path: string, ticket: number) =>
  invoke<{ text: string; summaryPath: string }>("summarize_transcript", { path, ticket });

export const answerQuestion = (question: string, context: string, path: string) =>
  invoke<{ text: string; summaryPath: string }>("answer_question", {
    question,
    context,
    path,
  });
