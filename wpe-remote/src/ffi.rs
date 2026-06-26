use std::os::raw::{c_char, c_int, c_uint, c_void};

pub enum WpeViewBackend {}
pub enum WpeViewBackendExportableFdo {}
pub enum WpeFdoShmExportedBuffer {}
pub enum WlShmBuffer {}
pub enum WlResource {}
pub enum WebKitWebView {}
pub enum WebKitWebViewBackend {}

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

/// From /usr/include/wpe-1.0/wpe/input.h
#[repr(C)]
pub struct WpeInputKeyboardEvent {
    pub time:              u32,
    pub key_code:          u32,  // XKB keysym
    pub hardware_key_code: u32,
    pub pressed:           bool,
    pub modifiers:         u32,
}

// XKB keysyms
pub const XKB_KEY_RETURN:     u32 = 0xFF0D;
pub const XKB_KEY_ARROW_UP:   u32 = 0xFF52;
pub const XKB_KEY_ARROW_DOWN: u32 = 0xFF54;
pub const XKB_KEY_ARROW_LEFT: u32 = 0xFF51;
pub const XKB_KEY_ARROW_RIGHT:u32 = 0xFF53;

extern "C" {
    pub fn wpe_loader_init(impl_library_name: *const c_char) -> bool;
    pub fn wpe_fdo_initialize_shm() -> bool;

    pub fn wpe_view_backend_exportable_fdo_create(
        client: *const WpeViewBackendExportableFdoClient,
        data:   *mut c_void,
        width:  c_uint,
        height: c_uint,
    ) -> *mut WpeViewBackendExportableFdo;

    pub fn wpe_view_backend_exportable_fdo_destroy(exportable: *mut WpeViewBackendExportableFdo);

    pub fn wpe_view_backend_exportable_fdo_get_view_backend(
        exportable: *mut WpeViewBackendExportableFdo,
    ) -> *mut WpeViewBackend;

    pub fn wpe_view_backend_exportable_fdo_dispatch_frame_complete(
        exportable: *mut WpeViewBackendExportableFdo,
    );

    pub fn wpe_view_backend_exportable_fdo_dispatch_release_shm_exported_buffer(
        exportable: *mut WpeViewBackendExportableFdo,
        buffer:     *mut WpeFdoShmExportedBuffer,
    );

    pub fn wpe_fdo_shm_exported_buffer_get_shm_buffer(
        buffer: *mut WpeFdoShmExportedBuffer,
    ) -> *mut WlShmBuffer;

    /// Dispatch a keyboard event directly to the WPE view backend.
    pub fn wpe_view_backend_dispatch_keyboard_event(
        backend: *mut WpeViewBackend,
        event:   *const WpeInputKeyboardEvent,
    );
}

extern "C" {
    pub fn wl_shm_buffer_begin_access(buffer: *mut WlShmBuffer);
    pub fn wl_shm_buffer_end_access(buffer: *mut WlShmBuffer);
    pub fn wl_shm_buffer_get_data(buffer: *mut WlShmBuffer) -> *mut c_void;
    pub fn wl_shm_buffer_get_stride(buffer: *const WlShmBuffer) -> i32;
    pub fn wl_shm_buffer_get_width(buffer: *const WlShmBuffer) -> i32;
    pub fn wl_shm_buffer_get_height(buffer: *const WlShmBuffer) -> i32;
}

extern "C" {
    pub fn webkit_web_view_backend_new(
        backend:   *mut WpeViewBackend,
        notify:    Option<unsafe extern "C" fn(*mut c_void)>,
        user_data: *mut c_void,
    ) -> *mut WebKitWebViewBackend;

    pub fn webkit_web_view_new(backend: *mut WebKitWebViewBackend) -> *mut WebKitWebView;

    pub fn webkit_web_view_load_html(
        view:     *mut WebKitWebView,
        content:  *const c_char,
        base_uri: *const c_char,
    );

    pub fn g_object_unref(obj: *mut c_void);

    pub fn g_signal_connect_data(
        instance:        *mut c_void,
        detailed_signal: *const c_char,
        c_handler:       Option<unsafe extern "C" fn()>,
        data:            *mut c_void,
        destroy_data:    Option<unsafe extern "C" fn(*mut c_void, *mut c_void)>,
        connect_flags:   c_uint,
    ) -> u64;
}
