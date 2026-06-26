mod delegate;
mod stream;

use std::path::Path;
use std::rc::Rc;
use std::sync::Arc;
use std::time::Duration;

use clap::Parser;
use dpi::PhysicalSize;
use servo::{RenderingContext, ServoBuilder, SoftwareRenderingContext, WebViewBuilder};
use tokio::sync::broadcast;
use url::Url;

use delegate::StreamDelegate;

const DEMO_HTML: &str = r#"<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  * { margin: 0; box-sizing: border-box; }
  body {
    background: linear-gradient(135deg, #0d1117, #161b22);
    color: #e6edf3;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", monospace;
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100vh;
    flex-direction: column;
    gap: 24px;
  }
  .clock {
    font-size: 72px;
    font-weight: 300;
    letter-spacing: 8px;
    color: #58a6ff;
    font-variant-numeric: tabular-nums;
  }
  .label {
    font-size: 14px;
    color: #8b949e;
    letter-spacing: 2px;
    text-transform: uppercase;
  }
</style>
</head>
<body>
  <div class="label">servo-screencaster</div>
  <div class="clock" id="t">00:00:00</div>
  <div class="label" id="d"></div>
  <script>
    function update() {
      const now = new Date();
      document.getElementById('t').textContent =
        now.toTimeString().slice(0, 8);
      document.getElementById('d').textContent =
        now.toDateString();
    }
    update();
    setInterval(update, 1000);
  </script>
</body>
</html>"#;

#[derive(Parser, Debug)]
#[command(name = "servo-screencaster", about = "Render HTML and stream as MJPEG over HTTP")]
struct Args {
    /// HTML file to render (omit for built-in demo)
    #[arg(long)]
    html: Option<String>,

    /// HTTP port for the MJPEG stream
    #[arg(long, default_value_t = 8080)]
    port: u16,

    /// Frames per second
    #[arg(long, default_value_t = 10)]
    fps: u32,

    /// Viewport width in pixels
    #[arg(long, default_value_t = 1280)]
    width: u32,

    /// Viewport height in pixels
    #[arg(long, default_value_t = 720)]
    height: u32,
}

fn resolve_url(args: &Args) -> (Url, Option<std::fs::File>) {
    match &args.html {
        Some(path) => {
            let abs = Path::new(path)
                .canonicalize()
                .unwrap_or_else(|_| panic!("File not found: {path}"));
            (Url::from_file_path(&abs).expect("invalid path"), None)
        }
        None => {
            // Write demo HTML to a temp file
            let tmp_path = format!("/tmp/servo-screencaster-demo-{}.html", std::process::id());
            std::fs::write(&tmp_path, DEMO_HTML).expect("write demo HTML");
            let url = Url::from_file_path(&tmp_path).unwrap();
            eprintln!("Using built-in demo (live clock)");
            (url, None)
        }
    }
}

fn main() {
    let args = Args::parse();
    let size = PhysicalSize::new(args.width, args.height);

    let (tx, _) = broadcast::channel::<Arc<Vec<u8>>>(4);

    // Start MJPEG HTTP server on a background thread
    let port = args.port;
    {
        let tx = tx.clone();
        std::thread::spawn(move || {
            tokio::runtime::Runtime::new()
                .unwrap()
                .block_on(stream::serve(port, tx));
        });
    }

    let (url, _tmp) = resolve_url(&args);

    // Servo must run on the main thread (uses Rc internally)
    let rendering_context: Rc<dyn RenderingContext> = Rc::new(
        SoftwareRenderingContext::new(size).expect("SoftwareRenderingContext::new failed"),
    );
    let _ = rendering_context.make_current();

    let servo = ServoBuilder::default().build();

    let delegate = Rc::new(StreamDelegate::new(
        rendering_context.clone(),
        tx,
        args.fps,
        size,
    ));

    let _webview = WebViewBuilder::new(&servo, rendering_context)
        .url(url)
        .delegate(delegate.clone() as Rc<dyn servo::WebViewDelegate>)
        .build();

    eprintln!("Servo starting… open http://localhost:{}/", args.port);

    // Drive the Servo event loop forever
    loop {
        servo.spin_event_loop();
        std::thread::sleep(Duration::from_millis(1));
    }
}
