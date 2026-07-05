//! L34 Update: self-update flow seed.
//!
//! Reads CURRENT_VERSION from env, queries the GitHub releases API for the
//! latest tag, and prints "up to date" or "outdated: <new>".

use std::env;

const GITHUB_API: &str = "https://api.github.com/repos/KooshaPari/nanovms/releases/latest";

fn current_version() -> String {
    env::var("CURRENT_VERSION").unwrap_or_else(|_| env!("CARGO_PKG_VERSION").to_string())
}

fn parse_latest_tag(body: &str) -> Option<String> {
    let needle = "\"tag_name\":";
    let idx = body.find(needle)?;
    let rest = &body[idx + needle.len()..];
    let start = rest.find('"')? + 1;
    let end_rel = rest[start..].find('"')?;
    Some(rest[start..start + end_rel].to_string())
}

fn is_newer(latest: &str, current: &str) -> bool {
    let parse = |s: &str| -> Vec<u32> {
        s.trim_start_matches('v').split('.').filter_map(|p| p.parse().ok()).collect()
    };
    let l = parse(latest);
    let c = parse(current);
    l > c
}

fn main() {
    let current = current_version();
    eprintln!("[update] current version: {}", current);
    let resp = ureq::get(GITHUB_API).call();
    match resp {
        Ok(r) => {
            let body = r.into_string().unwrap_or_default();
            if let Some(tag) = parse_latest_tag(&body) {
                if is_newer(&tag, &current) {
                    println!("outdated: {}", tag);
                } else {
                    println!("up to date");
                }
            } else {
                eprintln!("[update] could not parse latest tag");
                std::process::exit(1);
            }
        }
        Err(e) => {
            eprintln!("[update] GitHub API error: {}", e);
            std::process::exit(2);
        }
    }
}
