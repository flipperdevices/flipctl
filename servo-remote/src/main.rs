mod delegate;
mod input;
mod stats;
mod stream;

use std::path::Path;
use std::rc::Rc;
use std::sync::Arc;
use std::time::Duration;

use clap::Parser;
use dpi::PhysicalSize;

use input::RemoteKey;
use servo::{
    InputEvent, Key, KeyState, KeyboardEvent, NamedKey,
    RenderingContext, ServoBuilder, SoftwareRenderingContext, WebViewBuilder,
};
use tokio::sync::{broadcast, mpsc};
use url::Url;

use delegate::StreamDelegate;

const DEMO_HTML: &str = include_str!("demo.html");

#[derive(Parser, Debug)]
#[command(name = "servo-remote", about = "Headless HTML renderer with 5-way remote input")]
struct Args {
    /// HTML file to render (omit for built-in demo)
    #[arg(long)]
    html: Option<String>,

    /// HTTP port
    #[arg(long, default_value_t = 8080)]
    port: u16,

    /// Frames per second
    #[arg(long, default_value_t = 30)]
    fps: u32,

    /// Viewport width
    #[arg(long, default_value_t = 1280)]
    width: u32,

    /// Viewport height
    #[arg(long, default_value_t = 720)]
    height: u32,
}

fn resolve_url(args: &Args) -> Url {
    match &args.html {
        Some(path) => {
            let abs = Path::new(path)
                .canonicalize()
                .unwrap_or_else(|_| panic!("File not found: {path}"));
            Url::from_file_path(&abs).expect("invalid path")
        }
        None => {
            let tmp = format!("/tmp/servo-remote-demo-{}.html", std::process::id());
            std::fs::write(&tmp, DEMO_HTML).expect("write demo html");
            eprintln!("Using built-in navigation demo");
            Url::from_file_path(&tmp).unwrap()
        }
    }
}

fn remote_key_to_servo(key: RemoteKey) -> Key {
    match key {
        RemoteKey::Up    => Key::Named(NamedKey::ArrowUp),
        RemoteKey::Down  => Key::Named(NamedKey::ArrowDown),
        RemoteKey::Left  => Key::Named(NamedKey::ArrowLeft),
        RemoteKey::Right => Key::Named(NamedKey::ArrowRight),
        RemoteKey::Enter => Key::Named(NamedKey::Enter),
    }
}

fn main() {
    let args = Args::parse();
    let size = PhysicalSize::new(args.width, args.height);

    let (frame_tx, _) = broadcast::channel::<Arc<Vec<u8>>>(4);
    let (key_tx, mut key_rx) = mpsc::channel::<RemoteKey>(32);

    // Pixel channel: SyncSender(1) drops stale frame if encoder is busy
    let (pixel_tx, pixel_rx) = std::sync::mpsc::sync_channel::<image::RgbaImage>(1);

    // MJPEG + input HTTP server on background thread
    {
        let frame_tx = frame_tx.clone();
        let port = args.port;
        let (w, h) = (args.width, args.height);
        std::thread::spawn(move || {
            tokio::runtime::Runtime::new()
                .unwrap()
                .block_on(stream::serve(port, frame_tx, key_tx, w, h));
        });
    }

    // PNG encoding thread: RGBA pixels → PNG → broadcast
    {
        let frame_tx = frame_tx.clone();
        let (w, h) = (args.width, args.height);
        std::thread::spawn(move || {
            while let Ok(rgba) = pixel_rx.recv() {
                let t = std::time::Instant::now();
                let mut buf = Vec::new();
                use image::{ImageEncoder, ExtendedColorType};
                use image::codecs::png::{PngEncoder, CompressionType, FilterType};
                let enc = PngEncoder::new_with_quality(
                    &mut buf,
                    CompressionType::Fast,
                    FilterType::Sub,
                );
                if enc.write_image(rgba.as_raw(), w, h, ExtendedColorType::Rgba8).is_ok() {
                    stats::record_encode(t.elapsed().as_micros() as u64);
                    stats::inc_sent();
                    let _ = frame_tx.send(Arc::new(buf));
                }
            }
        });
    }

    // Stats printer thread: every 5 seconds
    std::thread::spawn(|| {
        let interval = 5.0f64;
        loop {
            std::thread::sleep(Duration::from_secs_f64(interval));
            stats::print_and_reset(interval);
        }
    });

    let url = resolve_url(&args);

    let rendering_context: Rc<dyn RenderingContext> = Rc::new(
        SoftwareRenderingContext::new(size).expect("SoftwareRenderingContext::new failed"),
    );
    let _ = rendering_context.make_current();

    let servo = ServoBuilder::default().build();

    let delegate = Rc::new(StreamDelegate::new(
        rendering_context.clone(),
        pixel_tx,
        args.fps,
        size,
    ));

    let webview = WebViewBuilder::new(&servo, rendering_context)
        .url(url)
        .delegate(delegate.clone() as Rc<dyn servo::WebViewDelegate>)
        .build();

    // Focus the webview so keyboard events are delivered to the page
    webview.focus();
    stats::init();

    eprintln!("Viewer:  http://localhost:{}/", args.port);

    loop {
        let mut had_key = false;
        while let Ok(key) = key_rx.try_recv() {
            let servo_key = remote_key_to_servo(key);
            for state in [KeyState::Down, KeyState::Up] {
                let ev = KeyboardEvent::from_state_and_key(state, servo_key.clone());
                webview.notify_input_event(InputEvent::Keyboard(ev));
            }
            stats::key_dispatched();
            had_key = true;
        }

        servo.spin_event_loop();

        // After key injection, spin aggressively to flush Servo's internal pipeline
        if had_key {
            for _ in 0..10 {
                servo.spin_event_loop();
            }
        } else {
            std::thread::sleep(Duration::from_millis(1));
        }
    }
}
