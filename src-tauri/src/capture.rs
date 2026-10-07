use tauri::WebviewWindow;

/// Ask the OS compositor to leave this window out of screen capture.
///
/// The window stays visible on this computer. Capture APIs that honor the
/// request never receive its pixels, so Zoom, Teams, and Meet do not show it.
pub fn exclude_from_capture(window: &WebviewWindow) {
    // Windows: SetWindowDisplayAffinity(WDA_EXCLUDEFROMCAPTURE)
    // macOS: NSWindow.sharingType = NSWindowSharingNone
    let _ = window.set_content_protected(true);

    #[cfg(target_os = "windows")]
    protect_windows(window);

    #[cfg(target_os = "linux")]
    protect_linux(window);
}

#[cfg(target_os = "windows")]
fn protect_windows(window: &WebviewWindow) {
    use std::ffi::c_void;

    #[link(name = "user32")]
    extern "system" {
        fn SetWindowDisplayAffinity(hwnd: *mut c_void, affinity: u32) -> i32;
    }

    /// Windows 10 2004+. Capture APIs skip this HWND. The window is not painted
    /// into the shared frame, so there is no black box where it was.
    const WDA_EXCLUDEFROMCAPTURE: u32 = 0x0000_0011;

    let Ok(hwnd) = window.hwnd() else {
        return;
    };
    unsafe {
        SetWindowDisplayAffinity(hwnd.0, WDA_EXCLUDEFROMCAPTURE);
    }
}

#[cfg(target_os = "linux")]
fn protect_linux(window: &WebviewWindow) {
    // GNOME's compositor has no client request that omits one window from a
    // monitor capture. X11 has no equivalent of WDA_EXCLUDEFROMCAPTURE either.
    // Keep the window visible. set_content_protected is a no-op here.
    let _ = window;
}
