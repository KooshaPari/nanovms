//! `dump-schema` — emit `odin.nvms` v0.2 JSON Schema to stdout (or a file).
//!
//! Usage:
//!
//! ```text
//! # Default: write to ../../schemas/odin.nvms.schema.json
//! cargo run --bin dump-schema
//!
//! # Or to stdout:
//! cargo run --bin dump-schema -- --stdout
//!
//! # Or to a custom path:
//! cargo run --bin dump-schema -- --out path/to/schema.json
//! ```
//!
//! The schema is generated from the Rust types at compile time via
//! `schemars`, so the JSON Schema can never drift from the Rust struct
//! definitions as long as `dump-schema` is re-run whenever `schema.rs`
//! changes.

use std::io;
use std::path::PathBuf;

use phenotype_manifest::write_schema_json;

/// Default location of the committed schema artifact, relative to the
/// `crates/phenotype-manifest` crate root (where this binary lives).
const DEFAULT_OUT_RELATIVE: &str = "../../schemas/odin.nvms.schema.json";

fn main() -> io::Result<()> {
    let args: Vec<String> = std::env::args().collect();
    let stdout = args.iter().any(|a| a == "--stdout");
    let out_path = parse_out_arg(&args).unwrap_or_else(|| {
        PathBuf::from(env!("CARGO_MANIFEST_DIR")).join(DEFAULT_OUT_RELATIVE)
    });

    if stdout {
        let stdout = io::stdout();
        let mut handle = stdout.lock();
        return write_schema_json(&mut handle);
    }

    if let Some(parent) = out_path.parent() {
        std::fs::create_dir_all(parent)?;
    }
    let mut file = std::fs::File::create(&out_path)?;
    write_schema_json(&mut file)?;
    eprintln!(
        "wrote schema to {}",
        out_path
            .strip_prefix(env!("CARGO_MANIFEST_DIR"))
            .unwrap_or(&out_path)
            .display()
    );
    Ok(())
}

fn parse_out_arg(args: &[String]) -> Option<PathBuf> {
    let idx = args.iter().position(|a| a == "--out")?;
    args.get(idx + 1).map(PathBuf::from)
}