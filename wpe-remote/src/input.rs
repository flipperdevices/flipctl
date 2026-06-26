use serde::Deserialize;
use crate::ffi::*;

#[derive(Debug, Clone, Copy, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum RemoteKey {
    Up,
    Down,
    Left,
    Right,
    Enter,
}

#[derive(Debug, Deserialize)]
pub struct InputBody {
    pub key: RemoteKey,
}

impl RemoteKey {
    pub fn to_xkb_keysym(self) -> u32 {
        match self {
            RemoteKey::Up    => XKB_KEY_ARROW_UP,
            RemoteKey::Down  => XKB_KEY_ARROW_DOWN,
            RemoteKey::Left  => XKB_KEY_ARROW_LEFT,
            RemoteKey::Right => XKB_KEY_ARROW_RIGHT,
            RemoteKey::Enter => XKB_KEY_RETURN,
        }
    }
}
