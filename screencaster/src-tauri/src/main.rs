use std::{
    convert::Infallible,
    path::Path,
    sync::{
        atomic::{AtomicBool, Ordering},
        Arc,
    },
    time::Duration,
};

use axum::{
    body::Body, extract::State, http::header, response::IntoResponse, routing::get, Router,
};
use bytes::Bytes;
use clap::Parser;
use tokio::sync::broadcast;

// ── CLI ──────────────────────────────────────────────────────────────────────

#[derive(Parser)]
#[command(about = "Headless HTML renderer → MJPEG stream")]
struct Args {
    /// HTML file to render (omit to use built-in demo)
    #[arg(long)]
    html: Option<String>,

    /// HTTP port for the MJPEG stream
    #[arg(long, default_value = "8080")]
    port: u16,

    /// Frames per second (1–60)
    #[arg(long, default_value = "10")]
    fps: u32,

    /// Viewport width in pixels
    #[arg(long, default_value = "1280")]
    width: u32,

    /// Viewport height in pixels
    #[arg(long, default_value = "720")]
    height: u32,
}

// ── Default demo HTML ─────────────────────────────────────────────────────────

const DEFAULT_HTML: &str = r#"<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Screencaster</title>
  <style>
    * { margin:0; padding:0; box-sizing:border-box; }
    body {
      background: linear-gradient(135deg, #0f0c29, #302b63, #24243e);
      color:#fff;
      font-family: 'Segoe UI', system-ui, sans-serif;
      display:flex; flex-direction:column;
      align-items:center; justify-content:center;
      height:100vh; gap:1.5rem; overflow:hidden;
    }
    h1   { font-size:2.5rem; font-weight:300; letter-spacing:0.2em; opacity:0.85; }
    .clk { font-size:6rem; font-weight:700; font-variant-numeric:tabular-nums;
           text-shadow:0 0 40px rgba(130,100,255,0.8); }
    .hint{ opacity:0.35; font-size:0.85rem; letter-spacing:0.05em; }
  </style>
</head>
<body>
  <h1>SCREENCASTER</h1>
  <div class="clk" id="c">--:--:--</div>
  <div class="hint">pass --html your-file.html to load your UI</div>
  <script>
    const el = document.getElementById('c');
    const tick = () => el.textContent =
      new Date().toLocaleTimeString('en-US', {hour12:false});
    setInterval(tick, 1000); tick();
  </script>
</body>
</html>"#;

// ── HTTP: viewer + MJPEG stream ───────────────────────────────────────────────

// broadcast::Sender used as state so each handler can subscribe()
type Frame   = Arc<Vec<u8>>;
type FrameTx = broadcast::Sender<Frame>;

async fn viewer() -> impl IntoResponse {
    axum::response::Html(r#"<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8"><title>screencaster</title>
  <style>
    *{margin:0;padding:0;box-sizing:border-box;}
    body{background:#000;display:flex;align-items:center;
         justify-content:center;min-height:100vh;}
    img{max-width:100%;max-height:100vh;display:block;}
  </style>
</head>
<body><img src="/stream"></body>
</html>"#)
}

async fn stream_mjpeg(State(tx): State<FrameTx>) -> impl IntoResponse {
    const B: &str = "mjpeg_boundary";

    // Each HTTP client gets its own broadcast subscription
    let mut sub = tx.subscribe();

    let s = async_stream::stream! {
        loop {
            match sub.recv().await {
                Ok(jpeg) => {
                    let hdr = format!(
                        "--{B}\r\nContent-Type: image/jpeg\r\nContent-Length: {}\r\n\r\n",
                        jpeg.len()
                    );
                    yield Ok::<Bytes, Infallible>(Bytes::from(hdr));
                    yield Ok(Bytes::copy_from_slice(&jpeg));
                    yield Ok(Bytes::from_static(b"\r\n"));
                }
                // Receiver too slow — skip lagged frames, keep going
                Err(broadcast::error::RecvError::Lagged(_)) => continue,
                // Channel closed — stream ends
                Err(broadcast::error::RecvError::Closed) => break,
            }
        }
    };

    axum::response::Response::builder()
        .header(header::CONTENT_TYPE, format!("multipart/x-mixed-replace;boundary={B}"))
        .body(Body::from_stream(s))
        .unwrap()
}

// ── Snapshot: WebKit → Cairo → JPEG ──────────────────────────────────────────

#[cfg(target_os = "linux")]
fn take_snapshot(wkv: &webkit2gtk::WebView, tx: &broadcast::Sender<Frame>) {
    use gtk::prelude::WidgetExt;
    use webkit2gtk::WebViewExt;

    let width  = wkv.allocated_width();
    let height = wkv.allocated_height();
    if width <= 0 || height <= 0 { return; }

    let tx = tx.clone();
    wkv.snapshot(
        webkit2gtk::SnapshotRegion::Visible,
        webkit2gtk::SnapshotOptions::empty(),
        None::<&gtk::gio::Cancellable>,
        move |result| {
            let surface = match result {
                Ok(s)  => s,
                Err(e) => { eprintln!("snapshot: {e}"); return; }
            };

            // Render onto an RGB24 ImageSurface to access raw pixels
            let mut img = match cairo::ImageSurface::create(
                cairo::Format::Rgb24, width, height,
            ) {
                Ok(s)  => s,
                Err(e) => { eprintln!("cairo: {e}"); return; }
            };
            {
                let ctx = cairo::Context::new(&img).unwrap();
                ctx.set_source_surface(&surface, 0.0, 0.0).unwrap();
                ctx.paint().unwrap();
            }

            let w = width  as u32;
            let h = height as u32;
            let stride = img.stride() as usize;

            let jpeg = {
                let data = img.data().unwrap();
                // Cairo Rgb24 on LE: bytes are [B, G, R, X] per pixel
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
                image::codecs::jpeg::JpegEncoder::new_with_quality(&mut out, 80)
                    .encode(&rgb, w, h, image::ExtendedColorType::Rgb8)
                    .unwrap();
                out
            };

            let _ = tx.send(Arc::new(jpeg));
        },
    );
}

// ── Entry point ───────────────────────────────────────────────────────────────

fn main() {
    // WebKit snapshot() doesn't work under Wayland — force X11 backend when available.
    // On a headless server (no display at all) we'll need Xvfb.
    if std::env::var("DISPLAY").is_ok() {
        std::env::set_var("GDK_BACKEND", "x11");
    }

    let args = Args::parse();
    let fps  = args.fps.clamp(1, 60);

    // Resolve HTML content + base URI so relative assets work
    let (html, base_uri): (String, Option<String>) = match &args.html {
        Some(path) => {
            let p       = Path::new(path);
            let content = std::fs::read_to_string(p)
                .unwrap_or_else(|e| panic!("Cannot read {path}: {e}"));
            let dir = p.parent().unwrap_or(Path::new("."));
            let abs = dir.canonicalize().unwrap_or_else(|_| dir.to_path_buf());
            (content, Some(format!("file://{}/", abs.display())))
        }
        None => (DEFAULT_HTML.to_string(), None),
    };

    // Channel capacity 1 — viewers only care about the latest frame
    let (tx, _) = broadcast::channel::<Frame>(1);

    // Axum MJPEG server on a dedicated tokio thread
    {
        let port = args.port;
        let tx2  = tx.clone();
        std::thread::spawn(move || {
            tokio::runtime::Runtime::new().unwrap().block_on(async move {
                let app = Router::new()
                    .route("/",       get(viewer))
                    .route("/stream", get(stream_mjpeg))
                    .with_state(tx2);
                let listener = tokio::net::TcpListener::bind(format!("0.0.0.0:{port}"))
                    .await.unwrap();
                eprintln!("Stream:  http://localhost:{port}/");
                axum::serve(listener, app).await.unwrap();
            });
        });
    }

    let interval = Duration::from_millis(1000 / fps as u64);
    let vp_w = args.width  as f64;
    let vp_h = args.height as f64;

    tauri::Builder::default()
        .setup(move |app| {
            // Hidden window — no display server needed with GDK_BACKEND=offscreen
            let win = tauri::WebviewWindowBuilder::new(
                app,
                "main",
                tauri::WebviewUrl::App("index.html".into()),
            )
            .inner_size(vp_w, vp_h)
            .build()?;

            // Minimize so the window doesn't interrupt the user's desktop.
            // snapshot() requires a mapped (realized+shown) window — visible(false) breaks it.
            let _ = win.minimize();

            let tx2   = tx.clone();
            let html2 = html.clone();
            let uri2  = base_uri.clone();

            win.with_webview(move |pv| {
                #[cfg(target_os = "linux")]
                {
                    use webkit2gtk::{LoadEvent, WebViewExt};

                    let wk = pv.inner();

                    // Force software rendering — required for snapshot() without GPU/display
                    {
                        use webkit2gtk::{SettingsExt, WebViewExt as _};
                        let settings = webkit2gtk::Settings::new();
                        settings.set_hardware_acceleration_policy(
                            webkit2gtk::HardwareAccelerationPolicy::Never,
                        );
                        wk.set_settings(&settings);
                    }

                    let started  = Arc::new(AtomicBool::new(false));
                    let started2 = started.clone();
                    let wk2      = wk.clone();
                    let tx3      = tx2.clone();

                    // Start snapshot timer once, on first page-load
                    wk.connect_load_changed(move |_, event| {
                        if event == LoadEvent::Finished
                            && !started2.swap(true, Ordering::Relaxed)
                        {
                            let wk3 = wk2.clone();
                            let tx4 = tx3.clone();
                            glib::timeout_add_local(interval, move || {
                                take_snapshot(&wk3, &tx4);
                                glib::ControlFlow::Continue
                            });
                        }
                    });

                    // Load the user's HTML (overrides Tauri's initial page)
                    wk.load_html(&html2, uri2.as_deref());
                }
            })?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("failed to run screencaster");
}
