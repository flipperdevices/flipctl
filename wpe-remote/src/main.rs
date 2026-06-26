mod ffi;
mod input;
mod renderer;
mod stream;

use std::sync::Arc;
use clap::Parser;
use tokio::sync::{broadcast, mpsc};

const DEMO_HTML: &str = include_str!("demo.html");

#[derive(Parser)]
#[command(name = "wpe-remote")]
struct Args {
    /// HTML file to render
    #[arg(long)]
    html: Option<String>,

    #[arg(long, default_value_t = 8080)]
    port: u16,

    #[arg(long, default_value_t = 30)]
    fps: u32,

    #[arg(long, default_value_t = 1280)]
    width: u32,

    #[arg(long, default_value_t = 720)]
    height: u32,
}

fn main() {
    let args = Args::parse();

    let html = match &args.html {
        Some(path) => std::fs::read_to_string(path)
            .unwrap_or_else(|_| panic!("Cannot read {path}")),
        None => {
            eprintln!("Using built-in demo");
            DEMO_HTML.to_string()
        }
    };

    let (frame_tx, _) = broadcast::channel::<Arc<Vec<u8>>>(4);
    let (key_tx, key_rx) = mpsc::channel::<input::RemoteKey>(32);

    // HTTP server on background thread
    {
        let frame_tx = frame_tx.clone();
        let port = args.port;
        std::thread::spawn(move || {
            tokio::runtime::Runtime::new()
                .unwrap()
                .block_on(stream::serve(port, frame_tx, key_tx));
        });
    }

    // WPE renderer takes over the main thread (GLib main loop)
    renderer::start_renderer(html, args.width, args.height, args.fps, frame_tx, key_rx);
}
