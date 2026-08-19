//! L23 Release notes: parse commits since last tag, group by prefix.

use std::process::Command;

fn main() {
    let last_tag = String::from_utf8(
        Command::new("git")
            .args(["describe", "--tags", "--abbrev=0"])
            .output()
            .map(|o| o.stdout)
            .unwrap_or_default(),
    ).unwrap_or_default();
    let last_tag = last_tag.trim().to_string();
    let range = if last_tag.is_empty() { "HEAD".to_string() } else { format!("{}..HEAD", last_tag) };

    let out = Command::new("git")
        .args(["log", "--pretty=format:%s", &range])
        .output()
        .map(|o| String::from_utf8_lossy(&o.stdout).to_string())
        .unwrap_or_default();

    let mut feat = 0; let mut fix = 0; let mut chore = 0; let mut other = 0;
    for line in out.lines() {
        if line.starts_with("feat:") || line.starts_with("feat(") { feat += 1; }
        else if line.starts_with("fix:") || line.starts_with("fix(") { fix += 1; }
        else if line.starts_with("chore:") || line.starts_with("chore(") { chore += 1; }
        else { other += 1; }
    }

    println!("## Release summary (since {})\n", if last_tag.is_empty() { "the beginning".to_string() } else { last_tag });
    println!("- Features: {}", feat);
    println!("- Fixes: {}", fix);
    println!("- Chores: {}", chore);
    println!("- Other: {}", other);
}
