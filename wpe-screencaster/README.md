# wpe-screencaster

Рендерит HTML/CSS/JS страницу и стримит её как MJPEG видеопоток по HTTP.
Работает **без дисплея, без X11, без Wayland** — использует WPE WebKit с SHM backend.

## Зависимости

```bash
sudo pacman -S wpewebkit   # Arch / Manjaro
```

Уже установлены как зависимости: `libwpe`, `wpebackend-fdo`.

## Сборка

```bash
cd wpe-screencaster
cargo build --release
```

Бинарь: `target/release/wpe-screencaster`

## Запуск

### Встроенное демо (живые часы)

```bash
./target/release/wpe-screencaster
```

### Свой HTML файл

```bash
./target/release/wpe-screencaster --html /path/to/ui.html
```

### Все параметры

```bash
./target/release/wpe-screencaster \
  --html /path/to/ui.html \
  --port 8080 \
  --fps 10 \
  --width 1280 \
  --height 720
```

| Параметр | По умолчанию | Описание |
|---|---|---|
| `--html` | встроенное демо | HTML файл для рендеринга |
| `--port` | 8080 | HTTP порт стрима |
| `--fps` | 10 | Кадров в секунду (1–60) |
| `--width` | 1280 | Ширина viewport в пикселях |
| `--height` | 720 | Высота viewport в пикселях |

## Просмотр стрима

После запуска открой в браузере:

```
http://localhost:8080/
```

Или прямой MJPEG поток (работает в VLC, ffplay, любом браузере):

```
http://localhost:8080/stream
```

```bash
# VLC
vlc http://localhost:8080/stream

# ffplay
ffplay http://localhost:8080/stream

# Сохранить в файл через ffmpeg (30 секунд)
ffmpeg -i http://localhost:8080/stream -t 30 output.mp4
```

## На сервере без дисплея

Работает без каких-либо переменных окружения:

```bash
./target/release/wpe-screencaster --html ui.html --port 8080
```

Если WebKit sandbox не работает (нет user namespaces в ядре):

```bash
WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS=1 \
  ./target/release/wpe-screencaster --html ui.html
```

## Как устроено

```
main()
  │
  ├── wpe_loader_init("libWPEBackend-fdo-1.0.so")
  ├── wpe_fdo_initialize_shm()   ← рендеринг в RAM без GPU/дисплея
  ├── WebKitWebView (WPE)        ← рендерит HTML/CSS/JS
  │     └── load_html(...)
  │
  ├── SHM frame callback         ← кадр готов
  │     BGRA → RGB → JPEG
  │     broadcast::channel
  │
  └── Axum HTTP :8080
        /         → HTML viewer
        /stream   → MJPEG поток
```

GLib main loop обрабатывает события WebKit на главном треде.
Axum HTTP сервер работает на отдельном tokio треде.
