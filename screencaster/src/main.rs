use std::{convert::Infallible, sync::Arc, time::Duration};

use anyhow::Result;
use axum::{
    body::Body, extract::State, http::header, response::IntoResponse, routing::get, Router,
};
use bytes::Bytes;
use clap::Parser;
use tokio::sync::watch;

// ── CLI ──────────────────────────────────────────────────────────────────────

#[derive(Parser)]
#[command(about = "Render a URL using system WebKit and stream it as MJPEG over HTTP")]
struct Args {
    /// URL to render
    url: String,

    /// HTTP port for the stream
    #[arg(long, default_value = "8080")]
    port: u16,

    /// Frames per second (1–60)
    #[arg(long, default_value = "10")]
    fps: u32,

    /// Viewport width
    #[arg(long, default_value = "1280")]
    width: u32,

    /// Viewport height
    #[arg(long, default_value = "720")]
    height: u32,
}

// ── HTTP handlers ─────────────────────────────────────────────────────────────

type Frame   = Arc<Vec<u8>>;
type FrameRx = watch::Receiver<Option<Frame>>;

async fn viewer() -> impl IntoResponse {
    axum::response::Html(r#"<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>screencaster</title>
  <style>
    * { margin:0; padding:0; box-sizing:border-box; }
    body { background:#0d0d0d; display:flex; flex-direction:column;
           align-items:center; justify-content:center; min-height:100vh;
           font-family:monospace; color:#555; }
    img  { max-width:100%; max-height:90vh; display:block; border:1px solid #222; }
    p    { margin-top:10px; font-size:11px; }
    a    { color:#555; }
  </style>
</head>
<body>
  <img src="/stream" alt="MJPEG stream">
  <p>MJPEG · <a href="/stream">/stream</a></p>
</body>
</html>"#)
}

async fn stream_mjpeg(State(rx): State<FrameRx>) -> impl IntoResponse {
    const BOUNDARY: &str = "mjpeg_boundary";

    let s = async_stream::stream! {
        let mut rx = rx;
        loop {
            if rx.changed().await.is_err() { break; }
            let frame = rx.borrow().clone();
            if let Some(jpeg) = frame {
                let hdr = format!(
                    "--{BOUNDARY}\r\nContent-Type: image/jpeg\r\nContent-Length: {}\r\n\r\n",
                    jpeg.len()
                );
                yield Ok::<Bytes, Infallible>(Bytes::from(hdr));
                yield Ok(Bytes::copy_from_slice(&jpeg));
                yield Ok(Bytes::from_static(b"\r\n"));
            }
        }
    };

    axum::response::Response::builder()
        .header(
            header::CONTENT_TYPE,
            format!("multipart/x-mixed-replace;boundary={BOUNDARY}"),
        )
        .body(Body::from_stream(s))
        .unwrap()
}

// ── Screenshot: WebKit → Cairo → JPEG ────────────────────────────────────────

#[cfg(target_os = "linux")]
fn take_snapshot(wkv: &webkit2gtk::WebView, tx: watch::Sender<Option<Frame>>) {
    use gtk::prelude::WidgetExt;
    use webkit2gtk::WebViewExt;

    let width  = wkv.allocated_width();
    let height = wkv.allocated_height();
    if width <= 0 || height <= 0 { return; }

    // WebKit snapshot is async; callback fires on the GLib main loop
    wkv.snapshot(
        webkit2gtk::SnapshotRegion::Visible,
        webkit2gtk::SnapshotOptions::empty(),
        None::<&gtk::gio::Cancellable>,
        move |result| {
            let surface = match result {
                Ok(s)  => s,
                Err(e) => { eprintln!("snapshot error: {e}"); return; }
            };

            // Paint WebKit surface onto a plain RGB24 ImageSurface so we can read pixels
            let mut img = match cairo::ImageSurface::create(cairo::Format::Rgb24, width, height) {
                Ok(s)  => s,
                Err(e) => { eprintln!("cairo error: {e}"); return; }
            };
            {
                let ctx = cairo::Context::new(&img).unwrap();
                ctx.set_source_surface(&surface, 0.0, 0.0).unwrap();
                ctx.paint().unwrap();
            } // drop Context → surface data becomes accessible

            let w      = width  as u32;
            let h      = height as u32;
            let stride = img.stride() as usize;

            let jpeg = {
                let data = img.data().unwrap();
                // Cairo Rgb24 on LE: [B, G, R, X] per pixel (32-bit BGRX)
                let mut rgb = Vec::with_capacity((w * h * 3) as usize);
                for row in 0..(h as usize) {
                    for col in 0..(w as usize) {
                        let i = row * stride + col * 4;
                        rgb.push(data[i + 2]); // R
                        rgb.push(data[i + 1]); // G
                        rgb.push(data[i    ]); // B
                    }
                }

                let mut out = Vec::new();
                image::codecs::jpeg::JpegEncoder::new_with_quality(&mut out, 75)
                    .encode(&rgb, w, h, image::ExtendedColorType::Rgb8)
                    .unwrap();
                out
            };

            let _ = tx.send(Some(Arc::new(jpeg)));
        },
    );
}

// ── Entry point ───────────────────────────────────────────────────────────────

fn main() -> Result<()> {
    let args = Args::parse();
    let fps  = args.fps.clamp(1, 60);

    #[cfg(not(target_os = "linux"))]
    anyhow::bail!("This tool uses WebKitGTK and currently only runs on Linux.");

    #[cfg(target_os = "linux")]
    {
        use tao::{
            dpi::LogicalSize,
            event::{Event, WindowEvent},
            event_loop::{ControlFlow, EventLoop},
            window::WindowBuilder,
        };
        use wry::{WebViewBuilder, WebViewExtUnix};

        // watch::Sender is Send+Sync, safe to share across GLib callbacks and threads
        let (tx, rx) = watch::channel::<Option<Frame>>(None);

        // Axum HTTP server on a background tokio thread
        {
            let port = args.port;
            let rx2  = rx.clone();
            std::thread::spawn(move || {
                tokio::runtime::Runtime::new().unwrap().block_on(async move {
                    let app = Router::new()
                        .route("/",       get(viewer))
                        .route("/stream", get(stream_mjpeg))
                        .with_state(rx2);
                    let addr     = format!("0.0.0.0:{port}");
                    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
                    eprintln!("Stream:  http://localhost:{port}/");
                    axum::serve(listener, app).await.unwrap();
                });
            });
        }

        // GTK + tao event loop on the main thread
        gtk::init()?;

        let event_loop = EventLoop::new();
        let window = WindowBuilder::new()
            .with_title("screencaster")
            .with_inner_size(LogicalSize::new(args.width, args.height))
            .build(&event_loop)?;

        let webview = WebViewBuilder::new()
            .with_url(&args.url)
            .build(&window)?;

        eprintln!("Navigating to {}...", args.url);

        // Grab the underlying webkit2gtk::WebView and set up the capture loop.
        // GLib timeouts run on the GTK main loop (same thread) — no threading issues.
        let wkv      = webview.webview();
        let interval = Duration::from_millis(1000 / fps as u64);
        glib::timeout_add_local(interval, move || {
            take_snapshot(&wkv, tx.clone());
            glib::ControlFlow::Continue
        });

        event_loop.run(move |event, _, control_flow| {
            *control_flow = ControlFlow::Wait;
            if let Event::WindowEvent {
                event: WindowEvent::CloseRequested, ..
            } = event
            {
                *control_flow = ControlFlow::Exit;
            }
        });
    }

    #[cfg(not(target_os = "linux"))]
    Ok(())
}
