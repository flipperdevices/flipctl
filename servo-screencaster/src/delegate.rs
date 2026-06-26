use std::cell::{Cell, RefCell};
use std::rc::Rc;
use std::sync::Arc;
use std::time::{Duration, Instant};

use dpi::PhysicalSize;
use euclid::{Box2D, Point2D};
use image::DynamicImage;
use servo::{LoadStatus, RenderingContext, WebView, WebViewDelegate};
use tokio::sync::broadcast;

use crate::stream::Frame;

pub struct StreamDelegate {
    pub loaded: Cell<bool>,
    rendering_context: Rc<dyn RenderingContext>,
    tx: broadcast::Sender<Frame>,
    last_frame: RefCell<Instant>,
    min_gap: Duration,
    size: PhysicalSize<u32>,
}

impl StreamDelegate {
    pub fn new(
        rendering_context: Rc<dyn RenderingContext>,
        tx: broadcast::Sender<Frame>,
        fps: u32,
        size: PhysicalSize<u32>,
    ) -> Self {
        Self {
            loaded: Cell::new(false),
            rendering_context,
            tx,
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
        webview.paint();

        let now = Instant::now();
        if now.duration_since(*self.last_frame.borrow()) < self.min_gap {
            return;
        }
        *self.last_frame.borrow_mut() = now;

        let rect = Box2D::new(
            Point2D::new(0, 0),
            Point2D::new(self.size.width as i32, self.size.height as i32),
        );
        let Some(rgba) = self.rendering_context.read_to_image(rect) else { return };

        let rgb = DynamicImage::ImageRgba8(rgba).to_rgb8();
        let mut jpeg_buf = Vec::new();
        use image::ImageEncoder;
        let enc = image::codecs::jpeg::JpegEncoder::new_with_quality(&mut jpeg_buf, 80);
        if enc
            .write_image(rgb.as_raw(), self.size.width, self.size.height, image::ExtendedColorType::Rgb8)
            .is_ok()
        {
            let _ = self.tx.send(Arc::new(jpeg_buf));
        }
    }
}
