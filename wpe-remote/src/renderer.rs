use std::{
    ffi::CString,
    sync::{Arc, atomic::{AtomicU64, Ordering}},
    time::{Duration, Instant},
};

// Shared timing: microseconds since program start
static T0: std::sync::OnceLock<Instant> = std::sync::OnceLock::new();
static KEY_DISPATCH_US: AtomicU64 = AtomicU64::new(0);
static FRAME_COUNT: AtomicU64 = AtomicU64::new(0);

fn now_us() -> u64 {
    T0.get_or_init(Instant::now).elapsed().as_micros() as u64
}

pub fn init_timer() {
    T0.get_or_init(Instant::now); // call at startup so T0 != key-dispatch time
}

use tokio::sync::{broadcast, mpsc};

use crate::ffi::*;
use crate::input::RemoteKey;
use crate::stream::Frame;

// ── Raw pixel buffer sent from GLib thread → encoding thread ─────────────────

struct PixelBuf {
    pixels: Vec<u8>,
    width:  u32,
    height: u32,
    stride: u32,
}

// ── Per-callback state (GLib thread only) ────────────────────────────────────

struct CallbackState {
    exportable: *mut WpeViewBackendExportableFdo,
    pixel_tx:   std::sync::mpsc::SyncSender<PixelBuf>,
    last_frame: Instant,
    min_gap:    Duration,
}

unsafe impl Send for CallbackState {}

// ── SHM frame callback ────────────────────────────────────────────────────────

unsafe extern "C" fn on_shm_buffer(
    data: *mut std::os::raw::c_void,
    buf:  *mut WpeFdoShmExportedBuffer,
) {
    let s = &mut *(data as *mut CallbackState);

    let now = Instant::now();
    if now.duration_since(s.last_frame) >= s.min_gap {
        s.last_frame = now;

        let shm    = wpe_fdo_shm_exported_buffer_get_shm_buffer(buf);
        wl_shm_buffer_begin_access(shm);
        let w      = wl_shm_buffer_get_width(shm)  as u32;
        let h      = wl_shm_buffer_get_height(shm) as u32;
        let stride = wl_shm_buffer_get_stride(shm) as u32;
        let src    = wl_shm_buffer_get_data(shm) as *const u8;
        let size   = (h * stride) as usize;

        // Fast pixel copy only — no encoding here
        let mut pixels = vec![0u8; size];
        std::ptr::copy_nonoverlapping(src, pixels.as_mut_ptr(), size);
        wl_shm_buffer_end_access(shm);

        let _ = s.pixel_tx.try_send(PixelBuf { pixels, width: w, height: h, stride });
    }

    // ACK immediately — unblocks WPE to start rendering the next frame
    wpe_view_backend_exportable_fdo_dispatch_frame_complete(s.exportable);
    wpe_view_backend_exportable_fdo_dispatch_release_shm_exported_buffer(s.exportable, buf);
}

unsafe extern "C" fn on_load_changed(
    _view: *mut std::os::raw::c_void,
    event: std::os::raw::c_int,
    _data: *mut std::os::raw::c_void,
) {
    if event == 3 { eprintln!("Page loaded."); }
}

// ── Keyboard dispatch ─────────────────────────────────────────────────────────

unsafe fn dispatch_key(backend: *mut WpeViewBackend, key: RemoteKey) {
    let keysym = key.to_xkb_keysym();
    let press = WpeInputKeyboardEvent {
        time: 0, key_code: keysym, hardware_key_code: 0, pressed: true, modifiers: 0,
    };
    wpe_view_backend_dispatch_keyboard_event(backend, &press);
    let release = WpeInputKeyboardEvent { pressed: false, ..press };
    wpe_view_backend_dispatch_keyboard_event(backend, &release);
}

// ── Entry point ───────────────────────────────────────────────────────────────

pub fn start_renderer(
    html:    String,
    width:   u32,
    height:  u32,
    fps:     u32,
    tx:      broadcast::Sender<Frame>,
    key_rx:  mpsc::Receiver<RemoteKey>,
) -> ! {
    // Encoding thread: BGRA pixels → RGB → JPEG → broadcast
    // Runs independently from GLib so the GLib thread is never blocked by encoding
    let (pixel_tx, pixel_rx) = std::sync::mpsc::sync_channel::<PixelBuf>(1);
    {
        let tx = tx.clone();
        std::thread::spawn(move || {
            while let Ok(pb) = pixel_rx.recv() {
                // BGRA → RGBA (simple channel swap, cheaper than BGRA→RGB)
                let mut rgba = Vec::with_capacity((pb.width * pb.height * 4) as usize);
                for row in 0..pb.height as usize {
                    for col in 0..pb.width as usize {
                        let i = row * pb.stride as usize + col * 4;
                        rgba.push(pb.pixels[i + 2]); // R
                        rgba.push(pb.pixels[i + 1]); // G
                        rgba.push(pb.pixels[i    ]); // B
                        rgba.push(255);               // A (opaque)
                    }
                }

                let mut out = Vec::new();
                use image::{ImageEncoder, ExtendedColorType};
                use image::codecs::png::{PngEncoder, CompressionType, FilterType};
                let enc = PngEncoder::new_with_quality(
                    &mut out,
                    CompressionType::Fast, // zlib level 1 — fast encode
                    FilterType::Sub,       // delta filter — good for UI content
                );
                if enc.write_image(&rgba, pb.width, pb.height, ExtendedColorType::Rgba8).is_ok() {
                    let _ = tx.send(Arc::new(out));
                }
            }
        });
    }

    unsafe {
        let lib = CString::new("libWPEBackend-fdo-1.0.so").unwrap();
        assert!(wpe_loader_init(lib.as_ptr()), "wpe_loader_init failed");
        assert!(wpe_fdo_initialize_shm(), "wpe_fdo_initialize_shm failed");

        let state = Box::new(CallbackState {
            exportable: std::ptr::null_mut(),
            pixel_tx,
            last_frame: Instant::now(),
            min_gap: Duration::from_millis(1000 / fps.clamp(1, 60) as u64),
        });
        let state_ptr = Box::into_raw(state);

        let client = WpeViewBackendExportableFdoClient {
            export_buffer_resource: None,
            export_dmabuf_resource: None,
            export_shm_buffer:      Some(on_shm_buffer),
            _reserved0: None,
            _reserved1: None,
        };
        let exportable = wpe_view_backend_exportable_fdo_create(
            &client, state_ptr as *mut _, width, height,
        );
        assert!(!exportable.is_null());

        let view_backend = wpe_view_backend_exportable_fdo_get_view_backend(exportable);
        assert!(!view_backend.is_null());
        (*state_ptr).exportable = exportable;

        let wk_backend = webkit_web_view_backend_new(view_backend, None, std::ptr::null_mut());
        let webview    = webkit_web_view_new(wk_backend);
        assert!(!webview.is_null(),
            "webkit_web_view_new failed — try WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS=1");

        let sig = CString::new("load-changed").unwrap();
        g_signal_connect_data(
            webview as *mut _,
            sig.as_ptr(),
            Some(std::mem::transmute::<
                unsafe extern "C" fn(*mut std::os::raw::c_void, i32, *mut std::os::raw::c_void),
                unsafe extern "C" fn(),
            >(on_load_changed)),
            std::ptr::null_mut(), None, 0,
        );

        let html_c = CString::new(html).unwrap();
        webkit_web_view_load_html(webview, html_c.as_ptr(), std::ptr::null());

        // Poll key channel every 5ms in GLib main loop — dispatches keys immediately
        // without waiting for the next render frame
        let vb_ptr = view_backend as usize;
        let key_rx_ptr = Box::into_raw(Box::new(key_rx));
        glib::timeout_add_local(Duration::from_millis(5), move || {
            let rx = &mut *key_rx_ptr;
            while let Ok(key) = rx.try_recv() {
                dispatch_key(vb_ptr as *mut WpeViewBackend, key);
            }
            glib::ControlFlow::Continue
        });

        init_timer(); // ensure T0 is set before any key arrives
        eprintln!("WPE renderer started.");
        glib::MainLoop::new(None, false).run();

        drop(Box::from_raw(key_rx_ptr));
        drop(Box::from_raw(state_ptr));
        g_object_unref(webview as *mut _);
        wpe_view_backend_exportable_fdo_destroy(exportable);
    }
    unreachable!()
}
