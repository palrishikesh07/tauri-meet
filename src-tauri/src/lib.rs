mod capture;

use serde::{Deserialize, Serialize};
use std::io::{BufRead, BufReader, Read, Write};
#[cfg(unix)]
use std::os::unix::process::CommandExt;
use std::path::PathBuf;
use std::process::{Child, Command, Stdio};
use std::sync::{mpsc, Arc, Mutex};
use std::time::{Duration, Instant};
use std::{fs, thread};
use tauri::{Emitter, Manager};

struct LiveText {
    text: String,
    partial: String,
}

struct Job {
    child: Child,
    path: String,
    text_path: String,
    source: String,
    started: Instant,
    live: Arc<Mutex<LiveText>>,
}

struct AppState {
    job: Mutex<Option<Job>>,
    ai_pid: Mutex<Option<u32>>,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct Device {
    id: String,
    label: String,
    detail: String,
    is_default: bool,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct DeviceList {
    downloads_dir: String,
    default_source: String,
    devices: Vec<Device>,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct RecordingFile {
    name: String,
    path: String,
    bytes: u64,
    modified: String,
    #[serde(default)]
    text_path: String,
    #[serde(default)]
    summary_path: String,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct ReadyMessage {
    status: String,
    path: String,
    source: String,
    #[serde(default)]
    text_path: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct RecordingStatus {
    recording: bool,
    path: Option<String>,
    text_path: Option<String>,
    source: Option<String>,
    elapsed_secs: u64,
    transcript: String,
    partial: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct SavedTake {
    path: String,
    bytes: u64,
    text_path: String,
    transcript: String,
}

fn engine_binary(
    app: &tauri::AppHandle,
) -> Result<PathBuf, String> {
    let engine_name = if cfg!(windows) {
        "meetrec.exe"
    } else {
        "meetrec"
    };

    if let Ok(path) = app
        .path()
        .resolve(
            engine_name,
            tauri::path::BaseDirectory::Resource,
        )
    {
        if path.is_file() {
            return Ok(path);
        }
    }

    let dev = PathBuf::from(
        env!("CARGO_MANIFEST_DIR"),
    )
    .join("../engine")
    .join(engine_name);

    if dev.is_file() {
        return Ok(dev);
    }

    Err(format!(
        "Recording engine is missing. Expected engine/{engine_name}. Build it first."
    ))
}

fn run_engine(
    app: &tauri::AppHandle,
    args: &[String],
) -> Result<String, String> {
    let bin = engine_binary(app)?;

    let output = Command::new(bin)
        .args(args)
        .output()
        .map_err(|err| {
            format!(
                "could not start the recorder: {err}"
            )
        })?;

    if !output.status.success() {
        let err_text =
            String::from_utf8_lossy(
                &output.stderr,
            )
            .trim()
            .to_string();

        if err_text.is_empty() {
            return Err(
                "recorder command failed".into()
            );
        }

        return Err(err_text);
    }

    Ok(
        String::from_utf8_lossy(
            &output.stdout,
        )
        .to_string(),
    )
}

async fn spawn_off_ui<T: Send + 'static>(
    work: impl FnOnce() -> Result<T, String>
        + Send
        + 'static,
) -> Result<T, String> {
    match tauri::async_runtime::spawn_blocking(
        work,
    )
    .await
    {
        Ok(result) => result,
        Err(err) => Err(format!(
            "background task stopped: {err}"
        )),
    }
}

#[cfg(unix)]
fn interrupt(child: &mut Child) {
    let pid = child.id();

    if pid == 0 ||
        pid > i32::MAX as u32
    {
        return;
    }

    unsafe {
        libc::kill(
            pid as i32,
            libc::SIGINT,
        );
    }
}

#[cfg(windows)]
fn interrupt(child: &mut Child) {
    // The Windows Go recorder listens on stdin for "stop". It then tells
    // FFmpeg to quit cleanly, finalizes the OGG file, and runs Whisper.
    if let Some(stdin) = child.stdin.as_mut() {
        let _ = stdin.write_all(b"stop\n");
        let _ = stdin.flush();
    } else {
        // Fallback for an unexpected recorder process without piped stdin.
        let _ = child.kill();
    }
}

fn clear_ai_pid(state: &AppState, pid: u32) {
    let mut guard = state.ai_pid.lock().expect("AI process state");
    if *guard == Some(pid) {
        *guard = None;
    }
}

struct AiProcessGuard<'a> {
    state: &'a AppState,
    pid: u32,
}

impl Drop for AiProcessGuard<'_> {
    fn drop(&mut self) {
        clear_ai_pid(self.state, self.pid);
    }
}

fn set_ai_pid(state: &AppState, pid: u32) {
    *state.ai_pid.lock().expect("AI process state") = Some(pid);
}

#[tauri::command]
fn cancel_ai_request(
    app: tauri::AppHandle,
) -> Result<(), String> {
    let state = app.state::<AppState>();

    // Take the PID before killing the process. This also prevents a second
    // cancel request from accidentally targeting a newer AI request.
    let pid = state
        .ai_pid
        .lock()
        .expect("AI process state")
        .take();

    let Some(pid) = pid else {
        // The AI process may have finished between the button click and
        // this command. That is already a successful cancellation state.
        return Ok(());
    };

    if pid == 0 {
        return Ok(());
    }

    #[cfg(windows)]
    {
        // The AI request runs inside meetrec.exe. /T is important because
        // meetrec.exe can have child processes, and /F makes cancellation
        // immediate rather than waiting for Ollama generation to finish.
        let taskkill = std::env::var_os("WINDIR")
            .map(|root| {
                std::path::PathBuf::from(root)
                    .join("System32")
                    .join("taskkill.exe")
            })
            .filter(|path| path.is_file())
            .unwrap_or_else(|| {
                std::path::PathBuf::from("taskkill.exe")
            });

        let status = Command::new(&taskkill)
            .args([
                "/PID",
                &pid.to_string(),
                "/T",
                "/F",
            ])
            .status()
            .map_err(|err| {
                format!(
                    "could not start taskkill for AI process {pid}: {err}"
                )
            })?;

        if !status.success() {
            // The process may have completed just before taskkill ran.
            // Verify whether the PID still exists before reporting an error.
            let check = Command::new(&taskkill)
                .args([
                    "/FI",
                    &format!("PID eq {pid}"),
                    "/NH",
                ])
                .output();

            let still_running = check
                .ok()
                .map(|output| {
                    String::from_utf8_lossy(&output.stdout)
                        .contains(&pid.to_string())
                })
                .unwrap_or(false);

            if still_running {
                return Err(format!(
                    "could not stop AI process (PID {pid})"
                ));
            }
        }
    }

    #[cfg(unix)]
    unsafe {
        libc::kill(pid as i32, libc::SIGTERM);
    }

    Ok(())
}

fn stop_job(
    job: Job,
    wait: Duration,
) -> Result<SavedTake, String> {
    let Job {
        mut child,
        path,
        text_path,
        ..
    } = job;

    interrupt(&mut child);

    let started = Instant::now();

    let status = loop {
        match child.try_wait() {
            Ok(Some(status)) => break status,

            Ok(None)
                if started.elapsed() > wait =>
            {
                let _ = child.kill();
                let _ = child.wait();
                return Err(
                    "recording stop timed out while creating the transcript (Whisper may still be processing)"
                        .into(),
                );
            }

            Ok(None) => {
                thread::sleep(
                    Duration::from_millis(40),
                );
            }

            Err(err) => {
                return Err(format!(
                    "could not stop the recording: {err}"
                ));
            }
        }
    };

    if !status.success() {
        return Err(format!(
            "recorder failed while creating the transcript (exit status: {status})"
        ));
    }

    let meta = fs::metadata(&path)
        .map_err(|_| {
            format!(
                "recording stopped, but the file was not written: {path}"
            )
        })?;

    let transcript =
        fs::read_to_string(&text_path)
            .unwrap_or_default();

    Ok(SavedTake {
        path,
        bytes: meta.len(),
        text_path,
        transcript,
    })
}

#[tauri::command]
fn list_devices(
    app: tauri::AppHandle,
) -> Result<DeviceList, String> {
    let raw = run_engine(
        &app,
        &["devices".into()],
    )?;

    serde_json::from_str(&raw)
        .map_err(|err| {
            format!(
                "could not read devices: {err}"
            )
        })
}

#[tauri::command]
fn list_recordings(
    app: tauri::AppHandle,
) -> Result<Vec<RecordingFile>, String> {
    let raw = run_engine(
        &app,
        &["list".into()],
    )?;

    serde_json::from_str(&raw)
        .map_err(|err| {
            format!(
                "could not read recordings: {err}"
            )
        })
}

#[tauri::command]
fn recording_status(
    state: tauri::State<'_, AppState>,
) -> RecordingStatus {
    let mut guard =
        state.job.lock().expect(
            "recording state",
        );

    if let Some(job) = guard.as_mut() {
        match job.child.try_wait() {
            Ok(Some(_)) => {
                *guard = None;
            }

            Ok(None) => {
                let live =
                    job.live.lock().expect(
                        "transcript",
                    );

                return RecordingStatus {
                    recording: true,
                    path: Some(
                        job.path.clone(),
                    ),
                    text_path: Some(
                        job.text_path.clone(),
                    ),
                    source: Some(
                        job.source.clone(),
                    ),
                    elapsed_secs: job
                        .started
                        .elapsed()
                        .as_secs(),
                    transcript: live.text.clone(),
                    partial: live.partial.clone(),
                };
            }

            Err(_) => {
                *guard = None;
            }
        }
    }

    RecordingStatus {
        recording: false,
        path: None,
        text_path: None,
        source: None,
        elapsed_secs: 0,
        transcript: String::new(),
        partial: String::new(),
    }
}

#[tauri::command]
async fn start_recording(
    app: tauri::AppHandle,
    source: Option<String>,
) -> Result<ReadyMessage, String> {
    spawn_off_ui(
        move || {
            start_recording_blocking(
                app,
                source,
            )
        },
    )
    .await
}

fn start_recording_blocking(
    app: tauri::AppHandle,
    source: Option<String>,
) -> Result<ReadyMessage, String> {
    let state =
        app.state::<AppState>();

    {
        let guard =
            state.job.lock().expect(
                "recording state",
            );

        if guard.is_some() {
            return Err(
                "a recording is already running"
                    .into(),
            );
        }
    }

    let bin = engine_binary(&app)?;

    let mut args =
        vec!["record".to_string()];

    if let Some(source) =
        source.filter(|value| !value.is_empty())
    {
        args.push(
            "--source".into(),
        );

        args.push(source);
    }

    let mut command =
        Command::new(bin);

    command
        .args(&args)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());

    #[cfg(unix)]
    unsafe {
        command.pre_exec(|| {
            libc::prctl(
                libc::PR_SET_PDEATHSIG,
                libc::SIGINT,
            );

            Ok(())
        });
    }

    let mut child = command
        .spawn()
        .map_err(|err| {
            format!(
                "could not start the recorder: {err}"
            )
        })?;

    let stdout = child
        .stdout
        .take()
        .ok_or_else(|| {
            "recorder stdout was not available"
                .to_string()
        })?;

    let mut stderr = child
        .stderr
        .take()
        .ok_or_else(|| {
            "recorder stderr was not available"
                .to_string()
        })?;

    let errors =
        Arc::new(Mutex::new(String::new()));

    let errors_for_thread =
        Arc::clone(&errors);

    thread::spawn(move || {
        let mut text = String::new();

        let _ =
            stderr.read_to_string(&mut text);

        *errors_for_thread
            .lock()
            .expect("stderr") = text;
    });

    let live =
        Arc::new(Mutex::new(
            LiveText {
                text: String::new(),
                partial: String::new(),
            },
        ));

    let live_for_thread =
        Arc::clone(&live);

    let app_for_audio =
        app.clone();

    let (tx, rx) =
        mpsc::channel();

    thread::spawn(move || {
        let mut saw_ready = false;

        for line in
            BufReader::new(stdout).lines()
        {
            let Ok(line) = line else {
                continue;
            };

            if line.trim().is_empty() {
                continue;
            }

            if !saw_ready {
                saw_ready = true;

                let _ =
                    tx.send(line);

                continue;
            }

            let Ok(value) =
                serde_json::from_str::<
                    serde_json::Value,
                >(&line)
            else {
                continue;
            };

            match value
                .get("status")
                .and_then(|status| status.as_str())
            {
                Some("pcm") => {
                    if let Some(pcm) =
                        value
                            .get("pcm")
                            .and_then(|pcm| pcm.as_str())
                    {
                        let _ =
                            app_for_audio.emit(
                                "pcm",
                                pcm,
                            );
                    }
                }

                Some("text") => {
                    let text = value
                        .get("text")
                        .and_then(|text| {
                            text.as_str()
                        })
                        .unwrap_or("")
                        .to_string();

                    let partial = value
                        .get("partial")
                        .and_then(|partial| {
                            partial.as_str()
                        })
                        .unwrap_or("")
                        .to_string();

                    *live_for_thread
                        .lock()
                        .expect("transcript") =
                        LiveText {
                            text,
                            partial,
                        };
                }

                _ => {}
            }
        }
    });

    let line =
        match rx.recv_timeout(
            Duration::from_secs(90),
        ) {
            Ok(line) => line,

            Err(_) => {
                interrupt(&mut child);

                let _ =
                    child.wait();

                thread::sleep(
                    Duration::from_millis(150),
                );

                let detail =
                    errors
                        .lock()
                        .expect("stderr")
                        .trim()
                        .to_string();

                if detail.is_empty() {
                    return Err(
                        "recording did not start"
                            .into(),
                    );
                }

                return Err(detail);
            }
        };

    let ready: ReadyMessage =
        serde_json::from_str(&line)
            .map_err(|err| {
                format!(
                    "unexpected recorder message: {err}"
                )
            })?;

    if ready.status != "recording" ||
        ready.path.is_empty()
    {
        interrupt(&mut child);
        let _ = child.wait();

        return Err(
            "recorder did not confirm the file"
                .into(),
        );
    }

    let mut guard =
        state.job.lock().expect(
            "recording state",
        );

    if guard.is_some() {
        interrupt(&mut child);
        let _ = child.wait();

        return Err(
            "a recording is already running"
                .into(),
        );
    }

    *guard = Some(Job {
        child,
        path: ready.path.clone(),
        text_path: ready.text_path.clone(),
        source: ready.source.clone(),
        started: Instant::now(),
        live,
    });

    Ok(ready)
}

#[tauri::command]
fn save_openai_key(
    app: tauri::AppHandle,
    key: String,
) -> Result<(), String> {
    let bin = engine_binary(&app)?;

    let output = Command::new(bin)
        .args(["save-key"])
        .env(
            "MEET_OPENAI_KEY",
            key.trim(),
        )
        .output()
        .map_err(|err| {
            format!(
                "could not save the key: {err}"
            )
        })?;

    if !output.status.success() {
        let err_text =
            String::from_utf8_lossy(
                &output.stderr,
            )
            .trim()
            .to_string();

        if err_text.is_empty() {
            return Err(
                "could not save the key".into(),
            );
        }

        return Err(err_text);
    }

    Ok(())
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct KeyStatus {
    saved: bool,
}

#[tauri::command]
fn openai_key_status(
    app: tauri::AppHandle,
) -> Result<KeyStatus, String> {
    let raw = run_engine(
        &app,
        &["key-status".into()],
    )?;

    serde_json::from_str(&raw)
        .map_err(|err| {
            format!(
                "could not read the key status: {err}"
            )
        })
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct OpenAIResult {
    text: String,
    summary_path: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct StreamLine {
    #[serde(default)]
    delta: String,

    #[serde(default)]
    text: String,

    #[serde(default)]
    summary_path: String,

    #[serde(default)]
    done: bool,
}

#[derive(Clone, Serialize)]
struct AnswerDelta {
    ticket: u32,
    text: String,
}

fn summarize_streaming(
    app: tauri::AppHandle,
    path: String,
    ticket: u32,
    topic: String,
) -> Result<OpenAIResult, String> {
    let bin = engine_binary(&app)?;

    let mut child = Command::new(bin)
        .args([
            "summarize",
            "--file",
            &path,
            "--topic",
            &topic,
            "--stream",
        ])
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|err| {
            format!(
                "could not start the answer: {err}"
            )
        })?;

    let state = app.state::<AppState>();
    let ai_pid = child.id();
    set_ai_pid(&state, ai_pid);
    let _ai_guard = AiProcessGuard {
        state: &state,
        pid: ai_pid,
    };

    let stdout = child
        .stdout
        .take()
        .ok_or_else(|| {
            "answer output was not available"
                .to_string()
        })?;

    let mut stderr = child
        .stderr
        .take()
        .ok_or_else(|| {
            "answer errors were not available"
                .to_string()
        })?;

    let errors =
        Arc::new(Mutex::new(String::new()));

    let errors_for_thread =
        Arc::clone(&errors);

    thread::spawn(move || {
        let mut text = String::new();

        let _ =
            stderr.read_to_string(
                &mut text,
            );

        *errors_for_thread
            .lock()
            .expect("stderr") = text;
    });

    let mut result:
        Option<OpenAIResult> = None;

    for line in
        BufReader::new(stdout).lines()
    {
        let line = line.map_err(
            |err| {
                format!(
                    "could not read the answer: {err}"
                )
            },
        )?;

        let Ok(parsed) =
            serde_json::from_str::<StreamLine>(
                &line,
            )
        else {
            continue;
        };

        if !parsed.delta.is_empty() {
            let _ =
                app.emit(
                    "answer-delta",
                    AnswerDelta {
                        ticket,
                        text: parsed.delta,
                    },
                );
        }

        if parsed.done {
            result =
                Some(OpenAIResult {
                    text: parsed.text,
                    summary_path:
                        parsed.summary_path,
                });
        }
    }

    let status =
        child.wait().map_err(|err| {
            format!(
                "could not finish the answer: {err}"
            )
        })?;

    if !status.success() {
        let err_text =
            errors
                .lock()
                .expect("stderr")
                .trim()
                .to_string();

        if err_text.is_empty() {
            return Err(
                "OpenAI returned no answer"
                    .into(),
            );
        }

        return Err(err_text);
    }

    result.ok_or_else(|| {
        "OpenAI returned no answer".into()
    })
}

fn summarize_local_streaming(
    app: tauri::AppHandle,
    path: String,
    ticket: u32,
    topic: String,
) -> Result<OpenAIResult, String> {
    let bin = engine_binary(&app)?;

    let mut child = Command::new(bin)
        .args([
            "local-summarize",
            "--file",
            &path,
            "--topic",
            &topic,
            "--stream",
        ])
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|err| {
            format!(
                "could not start local AI: {err}"
            )
        })?;

    let state = app.state::<AppState>();
    let ai_pid = child.id();
    set_ai_pid(&state, ai_pid);
    let _ai_guard = AiProcessGuard {
        state: &state,
        pid: ai_pid,
    };

    let stdout = child
        .stdout
        .take()
        .ok_or_else(|| {
            "local AI output was not available"
                .to_string()
        })?;

    let mut stderr = child
        .stderr
        .take()
        .ok_or_else(|| {
            "local AI errors were not available"
                .to_string()
        })?;

    let errors =
        Arc::new(Mutex::new(String::new()));

    let errors_for_thread =
        Arc::clone(&errors);

    thread::spawn(move || {
        let mut text = String::new();

        let _ =
            stderr.read_to_string(&mut text);

        *errors_for_thread
            .lock()
            .expect("stderr") = text;
    });

    let mut result:
        Option<OpenAIResult> = None;

    for line in
        BufReader::new(stdout).lines()
    {
        let line = line.map_err(|err| {
            format!(
                "could not read local AI answer: {err}"
            )
        })?;

        let Ok(parsed) =
            serde_json::from_str::<StreamLine>(
                &line,
            )
        else {
            continue;
        };

        if !parsed.delta.is_empty() {
            let _ =
                app.emit(
                    "answer-delta",
                    AnswerDelta {
                        ticket,
                        text: parsed.delta,
                    },
                );
        }

        if parsed.done {
            result =
                Some(OpenAIResult {
                    text: parsed.text,
                    summary_path:
                        parsed.summary_path,
                });
        }
    }

    let status =
        child.wait().map_err(|err| {
            format!(
                "could not finish local AI: {err}"
            )
        })?;

    if !status.success() {
        let err_text =
            errors
                .lock()
                .expect("stderr")
                .trim()
                .to_string();

        if err_text.is_empty() {
            return Err(
                "local AI returned no answer"
                    .into(),
            );
        }

        return Err(err_text);
    }

    result.ok_or_else(|| {
        "local AI returned no answer".into()
    })
}

#[tauri::command]
async fn summarize_transcript(
    app: tauri::AppHandle,
    path: String,
    ticket: u32,
    topic: String,
) -> Result<OpenAIResult, String> {
    spawn_off_ui(
        move || {
            summarize_streaming(
                app,
                path,
                ticket,
                topic,
            )
        },
    )
    .await
}

#[tauri::command]
async fn summarize_transcript_local(
    app: tauri::AppHandle,
    path: String,
    ticket: u32,
    topic: String,
) -> Result<OpenAIResult, String> {
    spawn_off_ui(move || {
        summarize_local_streaming(
            app,
            path,
            ticket,
            topic,
        )
    })
    .await
}

#[tauri::command]
async fn answer_question(
    app: tauri::AppHandle,
    question: String,
    context: String,
    path: String,
) -> Result<OpenAIResult, String> {
    spawn_off_ui(move || {
        let raw = run_engine(
            &app,
            &[
                "answer".into(),
                "--text".into(),
                question,
                "--context".into(),
                context,
                "--file".into(),
                path,
            ],
        )?;

        serde_json::from_str(&raw)
            .map_err(|err| {
                format!(
                    "could not read the answer: {err}"
                )
            })
    })
    .await
}

#[tauri::command]
fn save_transcript(
    path: String,
    text: String,
) -> Result<(), String> {
    let file =
        std::path::Path::new(&path);

    let name = file
        .file_name()
        .and_then(|name| name.to_str())
        .unwrap_or("");

    let is_downloads_path =
        if cfg!(windows) {
            path.contains("\\Downloads\\") ||
            path.contains("/Downloads/")
        } else {
            path.contains("/Downloads/")
        };

    if !is_downloads_path ||
        !name.starts_with("meet-") ||
        !name.ends_with(".txt")
    {
        return Err(
            "transcript path is not in Downloads"
                .into(),
        );
    }

    fs::write(file, text).map_err(
        |err| {
            format!(
                "could not save the transcript: {err}"
            )
        },
    )
}

#[tauri::command]
async fn stop_recording(
    app: tauri::AppHandle,
) -> Result<SavedTake, String> {
    let state =
        app.state::<AppState>();

    let job = state
        .job
        .lock()
        .expect("recording state")
        .take()
        .ok_or_else(|| {
            "nothing is recording".to_string()
        })?;

    spawn_off_ui(move || {
        stop_job(
            job,
            Duration::from_secs(180),
        )
    })
    .await
}

#[cfg_attr(
    mobile,
    tauri::mobile_entry_point
)]
pub fn run() {
    tauri::Builder::default()
        .manage(AppState {
            job: Mutex::new(None),
            ai_pid: Mutex::new(None),
        })
        .plugin(
            tauri_plugin_opener::init(),
        )
        .setup(|app| {
            if let Some(window) =
                app.get_webview_window("main")
            {
                capture::exclude_from_capture(
                    &window,
                );
            }

            Ok(())
        })
        .invoke_handler(
            tauri::generate_handler![
                list_devices,
                list_recordings,
                recording_status,
                start_recording,
                stop_recording,
                save_transcript,
                save_openai_key,
                openai_key_status,
                summarize_transcript,
                summarize_transcript_local,
                cancel_ai_request,
                answer_question
            ],
        )
        .build(
            tauri::generate_context!(),
        )
        .expect(
            "error while building tauri application",
        )
        .run(|_app, event| {
            match event {
                tauri::RunEvent::WindowEvent {
                    event: tauri::WindowEvent::CloseRequested { .. },
                    ..
                } => {
                    println!("Tauri: Window close requested");
                }

                tauri::RunEvent::ExitRequested { .. } => {
                    println!("Tauri: Application exit requested");
                }

                tauri::RunEvent::Exit => {
                    println!("Tauri: Application exited");
                }

                _ => {}
            }
        });
}