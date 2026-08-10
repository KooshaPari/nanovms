//! `phenotype-manifest` — v0.2 schema + validator + JSON Schema emission
//! for the `odin.nvms` manifest format shared by BytePort, OmniRoute,
//! and NanoVMS/PhenoCompose.
//!
//! Scope of this crate (v0.2):
//!
//! - [`Manifest`] — Rust struct with `serde::Deserialize` + `JsonSchema`
//!   covering the six top-level fields required by v0.2
//!   (`app`, `infra`, `network`, `observability`, `agent`, `portfolio`).
//! - [`validate`] — one-call `&str -> Result<Manifest, ManifestError>`
//!   using `serde_json` for structural validation and a small manual
//!   pass for semantic checks (e.g. `app.name` non-empty).
//! - [`schema_json`] / [`write_schema_json`] — emit the canonical JSON
//!   Schema (Draft 2020-12) for non-Rust consumers.
//!
//! Out of scope for v0.2 (deferred to v0.3):
//!
//! - Schema versioning / `version` discriminator on `Manifest`.
//! - YAML ingestion (JSON only for v0.2).
//! - Strict jsonschema-based validation in `validate()`
//!   (use [`schema_json`] downstream if you need it).
//!
//! # Quick start
//!
//! ```no_run
//! use phenotype_manifest::validate;
//! let json = r#"{
//!   "app":   {"name": "demo", "runtime": "node"},
//!   "infra": {"engine": "docker"}
//! }"#;
//! let m = validate(json).expect("minimal manifest");
//! assert_eq!(m.app.name, "demo");
//! ```

#![forbid(unsafe_code)]

mod schema;

pub use schema::{
    schema_json, validate, write_schema_json, Agent, App, Infra, Manifest, ManifestError, Network,
    Observability, Portfolio, Resources,
};
