mod input;
mod stream;

use std::{
    path::Path,
    sync::{
        atomic::{AtomicBool, Ordering},
        Arc,
    },
    time::Duration,
};

use clap::Parser;
use tokio::sync::{broadcast, mpsc};

use input::RemoteKey;
use stream::Frame;

const DEMO_HTML: &str = include_str!("demo.html");

#[derive(Parser)]
#[command(name = "tauri-remote", about = "Headless HTML renderer with 5-way remote input (Tauri/WebKitGTK)")]
struct Args {
    #[arg(long)]
    html: Option<String>,

    #[arg(long, default_value_t = 8080)]
    port: u16,

    #[arg(long, default_value_t = 10)]
    fps: u32,

    #[arg(long, default_value_t = 1280)]
    width: u32,

    #[arg(long, default_value_t = 720)]
    height: u32,
}

fn main() {
    // snapshot() requires X11 backend; force it when DISPLAY is available.
    if std::env::var("DISPLAY").is_ok() {
        std::env::set_var("GDK_BACKEND", "x11");
    }

    let args = Args::parse();
    let fps = args.fps.clamp(1, 60);

    let (html, base_uri): (String, Option<String>) = match &args.html {
        Some(path) => {
            let p = Path::new(path);
            let content = std::fs::read_to_string(p)
                .unwrap_or_else(|e| panic!("Cannot read {path}: {e}"));
            let dir = p.parent().unwrap_or(Path::new("."));
            let abs = dir.canonicalize().unwrap_or_else(|_| dir.to_path_buf());
            (content, Some(format!("file://{}/", abs.display())))
        }
        None => {
            eprintln!("Using built-in navigation demo");
            let tmp = format!("/tmp/tauri-remote-demo-{}.html", std::process::id());
            std::fs::write(&tmp, DEMO_HTML).expect("write demo html");
            let uri = format!("file://{tmp}");
            (DEMO_HTML.to_string(), Some(uri))
        }
    };

    let (frame_tx, _) = broadcast::channel::<Frame>(4);
    let (key_tx, mut key_rx) = mpsc::channel::<RemoteKey>(32);

    // HTTP server thread
    {
        let frame_tx = frame_tx.clone();
        let port = args.port;
        std::thread::spawn(move || {
            tokio::runtime::Runtime::new()
                .unwrap()
                .block_on(stream::serve(port, frame_tx, key_tx));
        });
    }

    let interval = Duration::from_millis(1000 / fps as u64);
    let vp_w = args.width as f64;
    let vp_h = args.height as f64;

    tauri::Builder::default()
        .setup(move |app| {
            let win = tauri::WebviewWindowBuilder::new(
                app,
                "main",
                tauri::WebviewUrl::App("index.html".into()),
            )
            .inner_size(vp_w, vp_h)
            .title("tauri-remote")
            .build()?;

            let _ = win.minimize();

            let frame_tx2 = frame_tx.clone();
            let html2 = html.clone();
            let uri2 = base_uri.clone();

            win.with_webview(move |pv| {
                #[cfg(target_os = "linux")]
                {
                    use webkit2gtk::{LoadEvent, WebViewExt};

                    let wk = pv.inner();

                    // Force software rendering so snapshot() works without a GPU/compositor
                    {
                        use webkit2gtk::{SettingsExt, WebViewExt as _};
                        let settings = webkit2gtk::Settings::new();
                        settings.set_hardware_acceleration_policy(
                            webkit2gtk::HardwareAccelerationPolicy::Never,
                        );
                        wk.set_settings(&settings);
                    }

                    let started   = Arc::new(AtomicBool::new(false));
                    let started2  = started.clone();
                    let wk2       = wk.clone();
                    let frame_tx3 = frame_tx2.clone();

                    // Start snapshot timer once, on first page-load
                    wk.connect_load_changed(move |_, event| {
                        if event == LoadEvent::Finished
                            && !started2.swap(true, Ordering::Relaxed)
                        {
                            let wk3 = wk2.clone();
                            let tx3 = frame_tx3.clone();
                            glib::timeout_add_local(interval, move || {
                                take_snapshot(&wk3, &tx3);
                                glib::ControlFlow::Continue
                            });
                        }
                    });

                    // Key injection: poll mpsc every 5ms, dispatch via evaluate_javascript
                    let wk_key = wk.clone();
                    glib::timeout_add_local(Duration::from_millis(5), move || {
                        while let Ok(key) = key_rx.try_recv() {
                            let k = key.to_js_key();
                            let js = format!(
                                "document.dispatchEvent(new KeyboardEvent('keydown',\
                                {{key:'{k}',bubbles:true,cancelable:true}}));\
                                document.dispatchEvent(new KeyboardEvent('keyup',\
                                {{key:'{k}',bubbles:true,cancelable:true}}));"
                            );
                            wk_key.evaluate_javascript(
                                &js, None, None,
                                None::<&gtk::gio::Cancellable>,
                                |_| {},
                            );
                        }
                        glib::ControlFlow::Continue
                    });

                    wk.load_html(&html2, uri2.as_deref());
                }
            })?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("failed to run tauri-remote");
}

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
                // Cairo Rgb24 on LE: [B, G, R, X] per pixel
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
