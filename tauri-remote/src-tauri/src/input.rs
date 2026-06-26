use serde::Deserialize;

#[derive(Debug, Clone, Copy, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum RemoteKey { Up, Down, Left, Right, Enter }

impl RemoteKey {
    pub fn to_js_key(self) -> &'static str {
        match self {
            Self::Up    => "ArrowUp",
            Self::Down  => "ArrowDown",
            Self::Left  => "ArrowLeft",
            Self::Right => "ArrowRight",
            Self::Enter => "Enter",
        }
    }
}

#[derive(Deserialize)]
pub struct InputBody { pub key: RemoteKey }
