use std::sync::Arc;
use axum::{Router, extract::State, response::IntoResponse};
use tokio::sync::broadcast;

pub type Frame = Arc<Vec<u8>>;

#[derive(Clone)]
pub struct AppState {
    pub tx: broadcast::Sender<Frame>,
}

pub async fn serve(port: u16, tx: broadcast::Sender<Frame>) {
    let state = AppState { tx };
    let app = Router::new()
        .route("/", axum::routing::get(viewer_handler))
        .route("/stream", axum::routing::get(stream_handler))
        .with_state(state);

    let addr = format!("0.0.0.0:{port}");
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    eprintln!("Streaming at http://localhost:{port}/");
    axum::serve(listener, app).await.unwrap();
}

async fn viewer_handler() -> impl IntoResponse {
    axum::response::Html(r#"<!DOCTYPE html><html>
<head><meta charset="utf-8"><title>servo-screencaster</title>
<style>*{margin:0}body{background:#000;display:flex;align-items:center;justify-content:center;min-height:100vh}img{max-width:100%;max-height:100vh}</style>
</head><body><img src="/stream"></body></html>"#)
}

async fn stream_handler(State(state): State<AppState>) -> impl IntoResponse {
    use axum::body::Body;
    use axum::http::{header, Response};

    let mut rx = state.tx.subscribe();
    let boundary = "servo_mjpeg";

    let stream = async_stream::stream! {
        loop {
            match rx.recv().await {
                Ok(frame) => {
                    let header = format!(
                        "--{boundary}\r\nContent-Type: image/jpeg\r\nContent-Length: {}\r\n\r\n",
                        frame.len()
                    );
                    yield Ok::<_, std::convert::Infallible>(bytes::Bytes::from(header));
                    yield Ok(bytes::Bytes::from(frame.as_ref().clone()));
                    yield Ok(bytes::Bytes::from_static(b"\r\n"));
                }
                Err(broadcast::error::RecvError::Lagged(_)) => continue,
                Err(broadcast::error::RecvError::Closed) => break,
            }
        }
    };

    Response::builder()
        .header(header::CONTENT_TYPE, format!("multipart/x-mixed-replace;boundary={boundary}"))
        .header(header::CACHE_CONTROL, "no-cache")
        .body(Body::from_stream(stream))
        .unwrap()
}
