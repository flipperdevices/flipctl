# servo-remote: исследование производительности

## Что такое servo-remote

Headless HTML рендерер на базе Servo browser engine (crates.io v0.3).
Рендерит HTML/CSS/JS без дисплея и стримит результат через WebSocket.

---

## Измеренные показатели (debug сборка, 1280×720)

```
[stats] FPS in:30 out:19-20 | paint:0.4ms read:1.5ms encode:50ms
[key→frame] 25-28ms
```

### Breakdown

| Операция | Время | Что делает |
|---|---|---|
| `paint()` | 0.4ms | Servo compositor → пиксели в SoftwareRenderingContext |
| `read_to_image()` | 1.5ms | Копирует 3.7MB из framebuffer в Vec |
| PNG encode | **50ms** | pure Rust zlib, CompressionType::Fast + FilterType::Sub |
| key→frame | 25-28ms | От dispatch keyboard event до следующего кадра |

### Вывод

Servo рендерит быстро — paint+read всего 2ms. Узкое место — **PNG кодирование (50ms)**, которое ограничивает FPS out до ~20.

---

## Что пробовали

### 1. requestAnimationFrame → setInterval

**Проблема:** `requestAnimationFrame` не работает в Servo headless — нет vsync, рендер-цикл останавливался после первого кадра. Клавиши диспатчились, DOM менялся, но новый кадр никогда не рендерился.

**Фикс:** `setInterval(() => { _t.textContent = Date.now(); }, 33)` в HTML — работает без vsync.

### 2. MJPEG → WebSocket

**Проблема:** MJPEG в браузере (Chrome) буферизуется на 0.5-1 секунду, создавая видимый лаг.

**Фикс:** WebSocket + `createImageBitmap` + canvas. Каждый кадр — отдельное бинарное WS сообщение, без накопления. Viewer HTML подключается к `/ws`.

### 3. JPEG (pure Rust) → PNG

**Было:** `image::codecs::jpeg::JpegEncoder`, quality 80 → ~20ms/кадр, 25KB.

**Стало:** `PngEncoder::new_with_quality(Fast, Sub)` → ~50ms/кадр, 60KB.

Для UI-контента (меню, текст, solid colors) PNG даёт lossless качество, но encode медленнее.

### 4. JPEG с turbojpeg

turbojpeg (libjpeg-turbo): ~3ms encode, 25KB. Нужен `TURBOJPEG_SOURCE=pkg-config` и `features = ["image", "pkg-config"]`. Значительно снижает encode bottleneck.

### 5. Raw RGBA (без кодирования)

**Идея:** отправлять сырые пиксели через WebSocket, браузер рендерит через `ImageData.putImageData()`.

**Результат:**
- encode: 0ms ✅
- FPS out: 20-22 (не вырос до 30) ❌
- key→frame min: 5ms ✅ (было 25ms)

**Вывод:** FPS ограничен пропускной способностью — 3.7MB × 20fps = 74 MB/s. Браузер не успевает принимать и рендерить кадры быстрее ~22fps при таком объёме данных. Зато отзывчивость на клавиши улучшилась.

---

## Дальнейшие направления

| Идея | Ожидаемый эффект |
|---|---|
| turbojpeg вместо PNG | encode 50ms → 3ms, FPS out → 30 |
| 640×360 вместо 1280×720 | PNG encode ~12ms, FPS out → 30 |
| `evaluate_javascript` для ввода | обходит input pipeline, потенциально быстрее |
| Servo Preferences | отключить неиспользуемые фичи (WebXR, WebGL...) |
| raw RGBA + LZ4 | быстрое сжатие без SIMD, меньше bandwidth |

---

## Архитектура потоков

```
Servo main thread:
  spin_event_loop() → notify_new_frame_ready()
    → webview.paint()       [0.4ms]
    → read_to_image()       [1.5ms]
    → pixel_tx.try_send()   [non-blocking, drop if busy]

Encoding thread:
  pixel_rx.recv()
    → PNG encode            [50ms]
    → broadcast::send()

Tokio thread (Axum):
  GET /ws  → WebSocket → binary frames → browser
  POST /input → mpsc → Servo event loop → keyboard event
```

## Запуск

```bash
cd /home/asd/rust_src/servo-remote
cargo build --release
./target/release/servo-remote
# → http://localhost:8080/
```
