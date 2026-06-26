mod ffi;
mod renderer;
mod stream;

use std::{path::Path, sync::Arc};

use clap::Parser;
use tokio::sync::broadcast;

// ── CLI ───────────────────────────────────────────────────────────────────────

#[derive(Parser)]
#[command(about = "Headless HTML renderer → MJPEG stream (WPE WebKit, no display required)")]
struct Args {
    /// HTML file to render (omit for built-in demo)
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

// ── Default HTML ──────────────────────────────────────────────────────────────

const DEFAULT_HTML: &str = r#"<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8"><title>wpe-screencaster</title>
  <style>
    *{margin:0;padding:0;box-sizing:border-box;}
    body{background:linear-gradient(135deg,#0f0c29,#302b63,#24243e);
         color:#fff;font-family:system-ui,sans-serif;
         display:flex;flex-direction:column;align-items:center;
         justify-content:center;height:100vh;gap:1.5rem;}
    h1{font-size:2.5rem;font-weight:300;letter-spacing:.2em;opacity:.85;}
    .clk{font-size:6rem;font-weight:700;font-variant-numeric:tabular-nums;
         text-shadow:0 0 40px rgba(130,100,255,.8);}
    .hint{opacity:.35;font-size:.85rem;}
  </style>
</head>
<body>
  <h1>WPE SCREENCASTER</h1>
  <div class="clk" id="c">--:--:--</div>
  <div class="hint">headless · no display required · wpewebkit</div>
  <script>
    const el=document.getElementById('c');
    const tick=()=>el.textContent=new Date().toLocaleTimeString('en-US',{hour12:false});
    setInterval(tick,1000);tick();
  </script>
</body>
</html>"#;

// ── Helpers ───────────────────────────────────────────────────────────────────

fn load_html(path: Option<&str>) -> (String, Option<String>) {
    match path {
        Some(p) => {
            let content = std::fs::read_to_string(p)
                .unwrap_or_else(|e| panic!("Cannot read {p}: {e}"));
            let dir = Path::new(p).parent().unwrap_or(Path::new("."));
            let abs = dir.canonicalize().unwrap_or_else(|_| dir.to_path_buf());
            (content, Some(format!("file://{}/", abs.display())))
        }
        None => (DEFAULT_HTML.to_string(), None),
    }
}

// ── Entry point ───────────────────────────────────────────────────────────────

fn main() {
    let args = Args::parse();
    let (html, base_uri) = load_html(args.html.as_deref());

    // broadcast capacity 2: one slot for "current frame", one for in-flight
    let (tx, _) = broadcast::channel::<Arc<Vec<u8>>>(2);

    // Axum MJPEG server on a background tokio thread
    {
        let port = args.port;
        let tx2  = tx.clone();
        std::thread::spawn(move || {
            tokio::runtime::Runtime::new()
                .unwrap()
                .block_on(stream::serve(port, tx2));
        });
    }

    // WPE renderer + GLib main loop — takes over the main thread, never returns
    renderer::start_renderer(html, base_uri, args.width, args.height, args.fps, tx);
}
