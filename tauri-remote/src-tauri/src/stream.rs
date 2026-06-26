use std::sync::Arc;
use axum::{
    Router,
    extract::{State, ws::{WebSocketUpgrade, WebSocket, Message}},
    response::IntoResponse,
    Json,
    http::StatusCode,
};
use tokio::sync::{broadcast, mpsc};

use crate::input::{InputBody, RemoteKey};

pub type Frame = Arc<Vec<u8>>;

#[derive(Clone)]
pub struct AppState {
    pub frame_tx: broadcast::Sender<Frame>,
    pub key_tx:   mpsc::Sender<RemoteKey>,
}

pub async fn serve(port: u16, frame_tx: broadcast::Sender<Frame>, key_tx: mpsc::Sender<RemoteKey>) {
    let state = AppState { frame_tx, key_tx };
    let app = Router::new()
        .route("/",       axum::routing::get(viewer_handler))
        .route("/ws",     axum::routing::get(ws_handler))
        .route("/stream", axum::routing::get(mjpeg_handler))
        .route("/input",  axum::routing::post(input_handler))
        .with_state(state);

    let listener = tokio::net::TcpListener::bind(format!("0.0.0.0:{port}")).await.unwrap();
    eprintln!("Viewer: http://localhost:{port}/");
    eprintln!("Input:  POST http://localhost:{port}/input  {{\"key\":\"up\"}}");
    axum::serve(listener, app).await.unwrap();
}

async fn viewer_handler() -> impl IntoResponse {
    axum::response::Html(r#"<!DOCTYPE html><html>
<head><meta charset="utf-8"><title>tauri-remote</title>
<style>
  * { margin: 0; }
  body { background: #000; display: flex; flex-direction: column;
         align-items: center; justify-content: center; min-height: 100vh; gap: 12px; }
  canvas { max-width: 100%; max-height: 85vh; }
  .btns { display: flex; gap: 8px; }
  button { background: #1e293b; color: #e2e8f0; border: 1px solid #334155;
           padding: 8px 16px; border-radius: 6px; cursor: pointer; font-size: 14px; }
  button:hover { background: #334155; }
</style>
</head><body>
  <canvas id="c"></canvas>
  <div class="btns">
    <button onclick="send('up')">&#9650;</button>
    <button onclick="send('left')">&#9664;</button>
    <button onclick="send('enter')">OK</button>
    <button onclick="send('right')">&#9654;</button>
    <button onclick="send('down')">&#9660;</button>
  </div>
  <script>
    const canvas = document.getElementById('c');
    const ctx = canvas.getContext('2d');

    const ws = new WebSocket('ws://' + location.host + '/ws');
    ws.binaryType = 'arraybuffer';
    let drawing = false;
    ws.onmessage = async (e) => {
      if (typeof e.data === 'string') return;
      if (drawing) return;
      drawing = true;
      try {
        const blob = new Blob([e.data], { type: 'image/jpeg' });
        const bmp = await createImageBitmap(blob);
        if (canvas.width !== bmp.width)   canvas.width  = bmp.width;
        if (canvas.height !== bmp.height) canvas.height = bmp.height;
        ctx.drawImage(bmp, 0, 0);
        bmp.close();
      } finally {
        drawing = false;
      }
    };
    ws.onerror = () => {
      canvas.outerHTML = '<img src="/stream" style="max-width:100%;max-height:85vh">';
    };

    async function send(key) {
      fetch('/input', { method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ key }) });
    }
    document.addEventListener('keydown', e => {
      const m = { ArrowUp:'up', ArrowDown:'down', ArrowLeft:'left', ArrowRight:'right', Enter:'enter' };
      if (m[e.key]) { e.preventDefault(); send(m[e.key]); }
    });
  </script>
</body></html>"#)
}

async fn ws_handler(
    ws: WebSocketUpgrade,
    State(state): State<AppState>,
) -> impl IntoResponse {
    let rx = state.frame_tx.subscribe();
    ws.on_upgrade(move |socket| ws_stream(socket, rx))
}

async fn ws_stream(mut socket: WebSocket, mut rx: broadcast::Receiver<Frame>) {
    loop {
        match rx.recv().await {
            Ok(frame) => {
                if socket.send(Message::Binary(frame.as_ref().clone())).await.is_err() {
                    break;
                }
            }
            Err(broadcast::error::RecvError::Lagged(_)) => continue,
            Err(broadcast::error::RecvError::Closed)   => break,
        }
    }
}

async fn mjpeg_handler(State(state): State<AppState>) -> impl IntoResponse {
    use axum::body::Body;
    use axum::http::{header, Response};
    let mut rx = state.frame_tx.subscribe();
    let boundary = "mjpeg_boundary";
    let stream = async_stream::stream! {
        loop {
            match rx.recv().await {
                Ok(frame) => {
                    let hdr = format!(
                        "--{boundary}\r\nContent-Type: image/jpeg\r\nContent-Length: {}\r\n\r\n",
                        frame.len()
                    );
                    yield Ok::<_, std::convert::Infallible>(bytes::Bytes::from(hdr));
                    yield Ok(bytes::Bytes::from(frame.as_ref().clone()));
                    yield Ok(bytes::Bytes::from_static(b"\r\n"));
                }
                Err(broadcast::error::RecvError::Lagged(_)) => continue,
                Err(broadcast::error::RecvError::Closed)   => break,
            }
        }
    };
    Response::builder()
        .header(header::CONTENT_TYPE, format!("multipart/x-mixed-replace;boundary={boundary}"))
        .header(header::CACHE_CONTROL, "no-cache")
        .body(Body::from_stream(stream))
        .unwrap()
}

async fn input_handler(
    State(state): State<AppState>,
    Json(body): Json<InputBody>,
) -> impl IntoResponse {
    let _ = state.key_tx.send(body.key).await;
    StatusCode::OK
}
