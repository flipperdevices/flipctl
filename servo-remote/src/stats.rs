use std::sync::atomic::{AtomicU64, Ordering};
use std::time::Instant;

// All durations stored as microseconds, counts as plain u64
static PAINT_US:    AtomicU64 = AtomicU64::new(0);
static READ_US:     AtomicU64 = AtomicU64::new(0);
static ENCODE_US:   AtomicU64 = AtomicU64::new(0);
static FRAMES_SENT: AtomicU64 = AtomicU64::new(0);
static FRAMES_PAINTED: AtomicU64 = AtomicU64::new(0);

// Key dispatch timestamp (us since T0) — set on dispatch, cleared after first frame
static KEY_DISPATCH_US: AtomicU64 = AtomicU64::new(0);
static T0: std::sync::OnceLock<Instant> = std::sync::OnceLock::new();

pub fn init() {
    T0.get_or_init(Instant::now);
}

fn now_us() -> u64 {
    T0.get_or_init(Instant::now).elapsed().as_micros() as u64
}

pub fn record_paint(us: u64)  { PAINT_US.fetch_add(us, Ordering::Relaxed); }
pub fn record_read(us: u64)   { READ_US.fetch_add(us, Ordering::Relaxed); }
pub fn record_encode(us: u64) { ENCODE_US.fetch_add(us, Ordering::Relaxed); }
pub fn inc_painted()          { FRAMES_PAINTED.fetch_add(1, Ordering::Relaxed); }
pub fn inc_sent()             { FRAMES_SENT.fetch_add(1, Ordering::Relaxed); }

pub fn key_dispatched() {
    KEY_DISPATCH_US.store(now_us(), Ordering::Relaxed);
}

/// Call at the start of each captured frame. Returns key→frame ms if a key was pending.
pub fn frame_captured() -> Option<u64> {
    let kd = KEY_DISPATCH_US.swap(0, Ordering::Relaxed);
    if kd > 0 {
        Some((now_us().saturating_sub(kd)) / 1000)
    } else {
        None
    }
}

/// Print and reset counters. Call every ~5 seconds.
pub fn print_and_reset(interval_secs: f64) {
    let painted = FRAMES_PAINTED.swap(0, Ordering::Relaxed);
    let sent    = FRAMES_SENT.swap(0, Ordering::Relaxed);
    let paint_us  = PAINT_US.swap(0, Ordering::Relaxed);
    let read_us   = READ_US.swap(0, Ordering::Relaxed);
    let encode_us = ENCODE_US.swap(0, Ordering::Relaxed);

    let fps_in  = painted as f64 / interval_secs;
    let fps_out = sent    as f64 / interval_secs;

    let avg = |total_us: u64, count: u64| -> f64 {
        if count == 0 { 0.0 } else { total_us as f64 / count as f64 / 1000.0 }
    };

    eprintln!(
        "[stats] FPS in:{fps_in:.1} out:{fps_out:.1} | \
         paint:{:.2}ms read:{:.2}ms encode:{:.2}ms",
        avg(paint_us,  painted),
        avg(read_us,   painted),
        avg(encode_us, sent),
    );
}
