use std::{convert::Infallible, sync::Arc};

use axum::{body::Body, extract::State, http::header, response::IntoResponse, routing::get, Router};
use bytes::Bytes;
use tokio::sync::broadcast;

pub type Frame   = Arc<Vec<u8>>;
pub type FrameTx = broadcast::Sender<Frame>;

async fn viewer() -> impl IntoResponse {
    axum::response::Html(r#"<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8"><title>wpe-screencaster</title>
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
                Err(broadcast::error::RecvError::Lagged(_)) => continue,
                Err(broadcast::error::RecvError::Closed)    => break,
            }
        }
    };

    axum::response::Response::builder()
        .header(header::CONTENT_TYPE, format!("multipart/x-mixed-replace;boundary={B}"))
        .body(Body::from_stream(s))
        .unwrap()
}

pub async fn serve(port: u16, tx: FrameTx) {
    let app = Router::new()
        .route("/",       get(viewer))
        .route("/stream", get(stream_mjpeg))
        .with_state(tx);
    let listener = tokio::net::TcpListener::bind(format!("0.0.0.0:{port}"))
        .await
        .unwrap_or_else(|e| panic!("Cannot bind :{port}: {e}"));
    eprintln!("Viewer:  http://localhost:{port}/");
    eprintln!("Stream:  http://localhost:{port}/stream");
    axum::serve(listener, app).await.unwrap();
}
