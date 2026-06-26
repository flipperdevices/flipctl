//! Raw FFI bindings for WPEBackend-fdo, WPEWebKit-2.0 and wayland-server.
//! All function signatures verified against installed system headers.

use std::os::raw::{c_char, c_int, c_uint, c_void};

// ── Opaque types ─────────────────────────────────────────────────────────────

pub enum WpeViewBackend {}
pub enum WpeViewBackendExportableFdo {}
pub enum WpeFdoShmExportedBuffer {}
pub enum WlShmBuffer {}
pub enum WlResource {}
pub enum WebKitWebView {}
pub enum WebKitWebViewBackend {}
pub enum WebKitSettings {}

// ── wpe_view_backend_exportable_fdo_client ───────────────────────────────────
// Layout verified from /usr/include/wpe-fdo-1.0/wpe/view-backend-exportable.h

#[repr(C)]
pub struct WpeDmabufResource {
    pub buffer_resource: *mut WlResource,
    pub width:     c_uint,
    pub height:    c_uint,
    pub format:    c_uint,
    pub n_planes:  u8,
    pub fds:       [c_int; 4],
    pub strides:   [c_uint; 4],
    pub offsets:   [c_uint; 4],
    pub modifiers: [u64; 4],
}

#[repr(C)]
pub struct WpeViewBackendExportableFdoClient {
    pub export_buffer_resource: Option<unsafe extern "C" fn(*mut c_void, *mut WlResource)>,
    pub export_dmabuf_resource: Option<unsafe extern "C" fn(*mut c_void, *mut WpeDmabufResource)>,
    pub export_shm_buffer:      Option<unsafe extern "C" fn(*mut c_void, *mut WpeFdoShmExportedBuffer)>,
    pub _reserved0: Option<unsafe extern "C" fn()>,
    pub _reserved1: Option<unsafe extern "C" fn()>,
}

// ── WPEBackend-fdo-1.0 ───────────────────────────────────────────────────────
// Sources:
//   wpe/unstable/initialize-shm.h  → wpe_fdo_initialize_shm
//   wpe/view-backend-exportable.h  → create / destroy / get_view_backend / dispatch_*
//   wpe/exported-buffer-shm.h      → wpe_fdo_shm_exported_buffer_get_shm_buffer

extern "C" {
    /// Load the WPE backend implementation library.
    /// Must be called FIRST, before any other WPE function.
    /// Pass "libWPEBackend-fdo-1.0.so" to use the FDO backend.
    pub fn wpe_loader_init(impl_library_name: *const c_char) -> bool;

    /// Initialize SHM rendering mode (no GPU/EGL required).
    /// Must be called before creating any WPE view backend.
    /// Returns false if FDO backend fails to set up its internal Wayland socket.
    pub fn wpe_fdo_initialize_shm() -> bool;

    pub fn wpe_view_backend_exportable_fdo_create(
        client: *const WpeViewBackendExportableFdoClient,
        data:   *mut c_void,
        width:  c_uint,
        height: c_uint,
    ) -> *mut WpeViewBackendExportableFdo;

    pub fn wpe_view_backend_exportable_fdo_destroy(
        exportable: *mut WpeViewBackendExportableFdo,
    );

    pub fn wpe_view_backend_exportable_fdo_get_view_backend(
        exportable: *mut WpeViewBackendExportableFdo,
    ) -> *mut WpeViewBackend;

    /// Tell WPE the frame has been consumed — schedules the next render.
    pub fn wpe_view_backend_exportable_fdo_dispatch_frame_complete(
        exportable: *mut WpeViewBackendExportableFdo,
    );

    /// Release the SHM buffer memory. Call AFTER dispatch_frame_complete.
    pub fn wpe_view_backend_exportable_fdo_dispatch_release_shm_exported_buffer(
        exportable: *mut WpeViewBackendExportableFdo,
        buffer:     *mut WpeFdoShmExportedBuffer,
    );

    /// Get the underlying Wayland SHM buffer for pixel access.
    pub fn wpe_fdo_shm_exported_buffer_get_shm_buffer(
        buffer: *mut WpeFdoShmExportedBuffer,
    ) -> *mut WlShmBuffer;
}

// ── wayland-server ────────────────────────────────────────────────────────────
// Signatures from /usr/include/wayland-server-core.h

extern "C" {
    /// Must be called before reading pixel data (protocol-level memory fence).
    pub fn wl_shm_buffer_begin_access(buffer: *mut WlShmBuffer);
    /// Must be called after reading pixel data.
    pub fn wl_shm_buffer_end_access(buffer: *mut WlShmBuffer);

    pub fn wl_shm_buffer_get_data(buffer: *mut WlShmBuffer) -> *mut c_void;
    pub fn wl_shm_buffer_get_stride(buffer: *const WlShmBuffer) -> i32;
    pub fn wl_shm_buffer_get_width(buffer: *const WlShmBuffer) -> i32;
    pub fn wl_shm_buffer_get_height(buffer: *const WlShmBuffer) -> i32;
}

// ── WPEWebKit-2.0 ─────────────────────────────────────────────────────────────
// Headers installed by wpewebkit package under /usr/include/wpe-webkit-2.0/

extern "C" {
    /// Wrap a raw WPE view backend into a WebKitWebViewBackend.
    /// notify/user_data are called when the wrapper is freed (pass NULL for both).
    pub fn webkit_web_view_backend_new(
        backend:   *mut WpeViewBackend,
        notify:    Option<unsafe extern "C" fn(*mut c_void)>,
        user_data: *mut c_void,
    ) -> *mut WebKitWebViewBackend;

    /// Create a WebView using the given backend wrapper.
    pub fn webkit_web_view_new(backend: *mut WebKitWebViewBackend) -> *mut WebKitWebView;

    /// Load HTML content with an optional base URI for resolving relative assets.
    pub fn webkit_web_view_load_html(
        view:     *mut WebKitWebView,
        content:  *const c_char,
        base_uri: *const c_char,
    );

    pub fn webkit_web_view_get_settings(view: *mut WebKitWebView) -> *mut WebKitSettings;
}

// GObject reference counting (from gobject-sys, already in dependency graph via glib)
extern "C" {
    pub fn g_object_unref(obj: *mut c_void);

    /// Connect a GObject signal.
    /// callback: bare `extern "C"` function pointer, cast to GCallback.
    /// data: passed as the last argument to the callback.
    pub fn g_signal_connect_data(
        instance:       *mut c_void,
        detailed_signal: *const c_char,
        c_handler:      Option<unsafe extern "C" fn()>,
        data:           *mut c_void,
        destroy_data:   Option<unsafe extern "C" fn(*mut c_void, *mut c_void)>,
        connect_flags:  c_uint,
    ) -> u64;
}
