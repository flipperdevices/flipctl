use std::{
    ffi::CString,
    sync::Arc,
    time::{Duration, Instant},
};

use tokio::sync::broadcast;

use crate::ffi::*;
use crate::stream::Frame;

// ── Per-callback state (lives on the GLib thread only) ───────────────────────

struct CallbackState {
    exportable: *mut WpeViewBackendExportableFdo,
    tx:         broadcast::Sender<Frame>,
    last_frame: Instant,
    min_gap:    Duration,
}

// SAFETY: CallbackState is only ever accessed from the single GLib main thread.
// We need this because Box<CallbackState> is stored as *mut c_void in C code,
// and Rust's Send analysis can't see that it never crosses threads.
unsafe impl Send for CallbackState {}

// ── BGRA → RGB conversion ─────────────────────────────────────────────────────

unsafe fn bgra_to_jpeg(
    shm:     *mut WlShmBuffer,
    quality: u8,
) -> Vec<u8> {
    wl_shm_buffer_begin_access(shm);

    let w      = wl_shm_buffer_get_width(shm)  as u32;
    let h      = wl_shm_buffer_get_height(shm) as u32;
    let stride = wl_shm_buffer_get_stride(shm) as usize;
    let src    = wl_shm_buffer_get_data(shm) as *const u8;

    // WL_SHM_FORMAT_ARGB8888 on LE = [B, G, R, A] per pixel
    let mut rgb = Vec::with_capacity((w * h * 3) as usize);
    for row in 0..(h as usize) {
        for col in 0..(w as usize) {
            let i = row * stride + col * 4;
            let px = std::slice::from_raw_parts(src.add(i), 4);
            rgb.push(px[2]); // R
            rgb.push(px[1]); // G
            rgb.push(px[0]); // B
        }
    }

    wl_shm_buffer_end_access(shm);

    let mut out = Vec::new();
    image::codecs::jpeg::JpegEncoder::new_with_quality(&mut out, quality)
        .encode(&rgb, w, h, image::ExtendedColorType::Rgb8)
        .unwrap_or_else(|e| eprintln!("JPEG encode: {e}"));
    out
}

// ── SHM frame callback ────────────────────────────────────────────────────────
// Must be an extern "C" fn — Rust closures can't be cast to C function pointers.

unsafe extern "C" fn on_shm_buffer(
    data: *mut std::os::raw::c_void,
    buf:  *mut WpeFdoShmExportedBuffer,
) {
    let s = &mut *(data as *mut CallbackState);

    let now = Instant::now();
    if now.duration_since(s.last_frame) >= s.min_gap {
        s.last_frame = now;

        let shm  = wpe_fdo_shm_exported_buffer_get_shm_buffer(buf);
        let jpeg = bgra_to_jpeg(shm, 80);
        let _    = s.tx.send(Arc::new(jpeg));
    }

    // ACK order matters: frame_complete first, then release (frees buffer memory)
    wpe_view_backend_exportable_fdo_dispatch_frame_complete(s.exportable);
    wpe_view_backend_exportable_fdo_dispatch_release_shm_exported_buffer(s.exportable, buf);
}

// ── GObject "load-changed" signal callback ────────────────────────────────────

unsafe extern "C" fn on_load_changed(
    _view:  *mut std::os::raw::c_void,
    event:  std::os::raw::c_int,
    _data:  *mut std::os::raw::c_void,
) {
    // WEBKIT_LOAD_FINISHED = 3
    if event == 3 {
        eprintln!("Page loaded.");
    }
}

// ── Public entry point ────────────────────────────────────────────────────────

pub fn start_renderer(
    html:     String,
    base_uri: Option<String>,
    width:    u32,
    height:   u32,
    fps:      u32,
    tx:       broadcast::Sender<Frame>,
) -> ! {
    unsafe {
        // 1. Load the FDO backend library — must be the very first WPE call
        let lib_name = CString::new("libWPEBackend-fdo-1.0.so").unwrap();
        assert!(
            wpe_loader_init(lib_name.as_ptr()),
            "wpe_loader_init failed — is wpebackend-fdo installed?"
        );

        // 2. Init SHM rendering mode (creates internal Wayland socket, no display needed)
        assert!(
            wpe_fdo_initialize_shm(),
            "wpe_fdo_initialize_shm() failed — check wpebackend-fdo installation"
        );

        // 2. Allocate callback state on the heap; pass raw pointer to FDO
        let state = Box::new(CallbackState {
            exportable: std::ptr::null_mut(), // filled in after create
            tx,
            last_frame: Instant::now(),
            min_gap:    Duration::from_millis(1000 / fps.clamp(1, 60) as u64),
        });
        let state_ptr = Box::into_raw(state);

        // 3. Create exportable view backend
        let client = WpeViewBackendExportableFdoClient {
            export_buffer_resource: None,
            export_dmabuf_resource: None,
            export_shm_buffer:      Some(on_shm_buffer),
            _reserved0: None,
            _reserved1: None,
        };
        let exportable = wpe_view_backend_exportable_fdo_create(
            &client,
            state_ptr as *mut _,
            width,
            height,
        );
        assert!(!exportable.is_null(), "wpe_view_backend_exportable_fdo_create failed");

        // Patch the state with the exportable pointer (needed inside the callback)
        (*state_ptr).exportable = exportable;

        // 4. Get the WPE view backend, wrap it, and create the WebKit WebView
        let wpe_backend = wpe_view_backend_exportable_fdo_get_view_backend(exportable);
        assert!(!wpe_backend.is_null(), "get_view_backend returned null");

        // webkit_web_view_backend_new wraps the raw wpe_view_backend for WebKit
        let wk_backend = webkit_web_view_backend_new(wpe_backend, None, std::ptr::null_mut());
        assert!(!wk_backend.is_null(), "webkit_web_view_backend_new returned null");

        let webview = webkit_web_view_new(wk_backend);
        assert!(!webview.is_null(), "webkit_web_view_new returned null\n\
            Hint: try WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS=1 if bubblewrap fails");

        // 5. Connect load-changed signal for status logging
        let signal_name = CString::new("load-changed").unwrap();
        g_signal_connect_data(
            webview as *mut _,
            signal_name.as_ptr(),
            Some(std::mem::transmute::<
                unsafe extern "C" fn(*mut std::os::raw::c_void, i32, *mut std::os::raw::c_void),
                unsafe extern "C" fn(),
            >(on_load_changed)),
            std::ptr::null_mut(),
            None,
            0,
        );

        // 6. Load the HTML
        let html_c    = CString::new(html).unwrap();
        let base_c    = base_uri.map(|u| CString::new(u).unwrap());
        let base_ptr  = base_c.as_ref().map_or(std::ptr::null(), |s| s.as_ptr());
        webkit_web_view_load_html(webview, html_c.as_ptr(), base_ptr);

        eprintln!("WPE WebView created, rendering started.");

        // 7. Run GLib main loop — processes WebKit events + our SHM callbacks
        // This never returns.
        glib::MainLoop::new(None, false).run();

        // Cleanup (unreachable in practice, but good form)
        g_object_unref(webview as *mut _);
        wpe_view_backend_exportable_fdo_destroy(exportable);
        drop(Box::from_raw(state_ptr));
    }

    unreachable!()
}
