//! L17 i18n runtime: `t(key)` returns the localized string from the
//! embedded locale table.

use std::collections::HashMap;
use std::sync::OnceLock;

fn table() -> &'static HashMap<String, String> {
    static TABLE: OnceLock<HashMap<String, String>> = OnceLock::new();
    TABLE.get_or_init(|| {
        let mut m = HashMap::new();
        m.insert("app.name".into(), "nanovms".into());
        m.insert("app.tagline".into(), "Cloud-runtime, microkernel-grade, anywhere.".into());
        m.insert("menu.file".into(), "File".into());
        m.insert("menu.edit".into(), "Edit".into());
        m.insert("menu.view".into(), "View".into());
        m.insert("menu.help".into(), "Help".into());
        m.insert("status.active".into(), "Active".into());
        m.insert("status.idle".into(), "Idle".into());
        m.insert("status.stale".into(), "Stale".into());
        m.insert("status.disconnected".into(), "Disconnected".into());
        m.insert("error.network".into(), "Network unavailable.".into());
        m.insert("error.permission".into(), "Permission denied.".into());
        m.insert("error.not_found".into(), "Resource not found.".into());
        m.insert("ok.saved".into(), "Saved successfully.".into());
        m.insert("ok.connected".into(), "Connected.".into());
        m
    })
}

pub fn t(key: &str) -> &'static str {
    table().get(key).map(String::as_str).unwrap_or("?")
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test] fn known_key() { assert_eq!(t("app.name"), "nanovms"); }
    #[test] fn unknown_key() { assert_eq!(t("nope"), "?"); }
}
