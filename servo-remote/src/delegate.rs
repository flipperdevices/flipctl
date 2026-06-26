use std::cell::{Cell, RefCell};
use std::rc::Rc;
use std::sync::mpsc::SyncSender;
use std::time::{Duration, Instant};

use dpi::PhysicalSize;
use euclid::{Box2D, Point2D};
use image::RgbaImage;
use servo::{LoadStatus, RenderingContext, WebView, WebViewDelegate};

use crate::stats;

pub struct StreamDelegate {
    pub loaded: Cell<bool>,
    rendering_context: Rc<dyn RenderingContext>,
    pixel_tx: SyncSender<RgbaImage>,
    last_frame: RefCell<Instant>,
    min_gap: Duration,
    size: PhysicalSize<u32>,
}

impl StreamDelegate {
    pub fn new(
        rendering_context: Rc<dyn RenderingContext>,
        pixel_tx: SyncSender<RgbaImage>,
        fps: u32,
        size: PhysicalSize<u32>,
    ) -> Self {
        Self {
            loaded: Cell::new(false),
            rendering_context,
            pixel_tx,
            last_frame: RefCell::new(Instant::now() - Duration::from_secs(1)),
            min_gap: Duration::from_millis(1000 / fps.max(1) as u64),
            size,
        }
    }
}

impl WebViewDelegate for StreamDelegate {
    fn notify_load_status_changed(&self, _webview: WebView, status: LoadStatus) {
        if matches!(status, LoadStatus::Complete) {
            self.loaded.set(true);
            eprintln!("Page loaded.");
        }
    }

    fn notify_new_frame_ready(&self, webview: WebView) {
        let t0 = Instant::now();
        webview.paint();
        stats::record_paint(t0.elapsed().as_micros() as u64);
        stats::inc_painted();

        let now = Instant::now();
        if now.duration_since(*self.last_frame.borrow()) < self.min_gap {
            return;
        }
        *self.last_frame.borrow_mut() = now;

        let rect = Box2D::new(
            Point2D::new(0, 0),
            Point2D::new(self.size.width as i32, self.size.height as i32),
        );

        let t1 = Instant::now();
        let Some(rgba) = self.rendering_context.read_to_image(rect) else { return };
        stats::record_read(t1.elapsed().as_micros() as u64);

        if let Some(latency_ms) = stats::frame_captured() {
            eprintln!("[key→frame] {latency_ms}ms");
        }

        let _ = self.pixel_tx.try_send(rgba);
    }
}
