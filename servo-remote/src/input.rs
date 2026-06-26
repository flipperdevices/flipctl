use serde::Deserialize;

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
