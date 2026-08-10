//! `odin.nvms` manifest schema — Rust types + validator for v0.2.
//!
//! The schema is the single source of truth for the manifest format.
//! Consuming repos (BytePort, OmniRoute, NanoVMS/PhenoCompose) validate
//! manifests against this crate at their respective trust boundaries.
//!
//! Two surfaces:
//!
//! - [`validate`] — parse-and-validate a JSON manifest string.
//! - [`schema_json`] — emit the canonical JSON Schema (Draft 07)
//!   derived from the Rust types via `schemars`. Ship the schema
//!   artifact to non-Rust consumers (BytePort Go, OmniRoute TS) so
//!   they validate against the same shape.

use schemars::JsonSchema;
use serde::{Deserialize, Serialize};
use thiserror::Error;

/// Errors produced by [`validate`].
#[derive(Debug, Error)]
pub enum ManifestError {
    /// The input could not be parsed as JSON, or did not match the v0.2
    /// schema shape (`deny_unknown_fields` is enabled).
    #[error("manifest JSON parse error: {0}")]
    Parse(#[from] serde_json::Error),

    /// The input parsed as JSON but failed a semantic validation rule.
    #[error("invalid manifest: {0}")]
    Invalid(String),
}

/// Top-level manifest object. Matches the `odin.nvms` v0.2 wire format.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, JsonSchema)]
#[serde(deny_unknown_fields)]
pub struct Manifest {
    /// Application identity & build configuration. Required.
    pub app: App,

    /// Infrastructure engine + resource envelope. Required.
    pub infra: Infra,

    /// Optional network configuration.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub network: Option<Network>,

    /// Optional observability configuration (OTel, log level).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub observability: Option<Observability>,

    /// Optional agent section (MCP tools / A2A skills).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub agent: Option<Agent>,

    /// Optional portfolio section.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub portfolio: Option<Portfolio>,
}

/// Application identity and build configuration.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, JsonSchema)]
#[serde(deny_unknown_fields)]
pub struct App {
    /// Human-readable app name (project slug, portfolio display name).
    /// Required, non-empty.
    pub name: String,

    /// Source repository URL. Optional.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub repo: Option<String>,

    /// Build command. Optional.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub build: Option<String>,

    /// Runtime identifier (e.g. `node`, `python`, `wasm`, `static`).
    /// Optional.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub runtime: Option<String>,
}

/// Infrastructure section — engine and its resource envelope.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, JsonSchema)]
#[serde(deny_unknown_fields)]
pub struct Infra {
    /// Execution engine id (e.g. `docker`, `firecracker`, `k8s`,
    /// `lambda`, `local`). Required, non-empty.
    pub engine: String,

    /// Optional resource envelope (CPU / memory).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub resources: Option<Resources>,
}

/// Resource envelope for the engine.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, JsonSchema)]
#[serde(deny_unknown_fields)]
pub struct Resources {
    /// CPU request (engine-specific unit; typically millicores for
    /// k8s-like engines). Kept as a free-form JSON value in v0.1 to
    /// avoid locking the wire shape to a single engine.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub cpu: Option<serde_json::Value>,

    /// Memory request (engine-specific unit; typically a string like
    /// `"512Mi"` or `"1Gi"`).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub memory: Option<String>,
}

/// Network section — ports and domains.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, JsonSchema)]
#[serde(deny_unknown_fields)]
pub struct Network {
    /// Public-facing ports.
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub ports: Vec<u16>,

    /// Custom domains.
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub domains: Vec<String>,
}

/// Observability section — OTel endpoint and log level.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, JsonSchema)]
#[serde(deny_unknown_fields)]
pub struct Observability {
    /// OTLP collector endpoint (e.g. `http://otel:4317`).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub otel_endpoint: Option<String>,

    /// Log level filter (`trace`/`debug`/`info`/`warn`/`error`).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub log_level: Option<String>,
}

/// Agent section — MCP tools + A2A skills enabled for this app.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, JsonSchema)]
#[serde(deny_unknown_fields)]
pub struct Agent {
    /// Enabled MCP tool names (e.g. `byteport_deploy`).
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub mcp_tools: Vec<String>,

    /// Exposed A2A skill names (e.g. `deploy_application`).
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub a2a_skills: Vec<String>,
}

/// Portfolio section — visibility and presentation.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, JsonSchema)]
#[serde(deny_unknown_fields)]
pub struct Portfolio {
    /// Visibility: `public`, `unlisted`, or `private`.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub visibility: Option<String>,

    /// Theme name.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub theme: Option<String>,
}

/// Parse and validate an `odin.nvms` manifest (JSON form).
///
/// Combines `serde_json` structural deserialization with a small
/// semantic pass. Returns the parsed [`Manifest`] on success.
///
/// # Errors
///
/// - [`ManifestError::Parse`] if the input is not valid JSON or does not
///   match the v0.2 schema shape (`deny_unknown_fields` is enabled).
/// - [`ManifestError::Invalid`] if a semantic rule is violated (e.g.
///   `app.name` is empty).
///
/// # Example
///
/// ```
/// use phenotype_manifest::validate;
/// let json = r#"{
///     "app": {"name": "demo"},
///     "infra": {"engine": "docker"}
/// }"#;
/// let manifest = validate(json).expect("minimal manifest");
/// assert_eq!(manifest.app.name, "demo");
/// ```
pub fn validate(json: &str) -> Result<Manifest, ManifestError> {
    let manifest: Manifest = serde_json::from_str(json)?;

    // Semantic rules (v0.2):
    if manifest.app.name.trim().is_empty() {
        return Err(ManifestError::Invalid("app.name must not be empty".into()));
    }
    if manifest.infra.engine.trim().is_empty() {
        return Err(ManifestError::Invalid(
            "infra.engine must not be empty".into(),
        ));
    }

    Ok(manifest)
}

/// Emit the canonical JSON Schema (Draft 07) for `odin.nvms` v0.2,
/// derived from the Rust types via `schemars`.
///
/// Use this to:
///
/// - Produce a stable schema artifact (`schemas/odin.nvms.schema.json`)
///   for non-Rust consumers (BytePort Go uses `gojsonschema`, OmniRoute
///   TS uses `ajv`).
/// - Bootstrap editor tooling (Cue, JSON Schema LSP, IDE intellisense).
///
/// The returned JSON is `pretty-printed` for human inspection and
/// committed to the workspace as a static artifact.
///
/// # Example
///
/// ```
/// use phenotype_manifest::schema_json;
/// let s = schema_json();
/// let v: serde_json::Value = serde_json::from_str(&s).unwrap();
/// assert_eq!(v["title"], "Manifest");
/// ```
pub fn schema_json() -> String {
    let schema = schemars::schema_for!(Manifest);
    serde_json::to_string_pretty(&schema).expect("schemars output is always JSON-serializable")
}

/// Same as [`schema_json`] but writes to a `Write` target.
///
/// Convenience for tooling that wants to stream the schema to a file or
/// stdout without round-tripping through a `String`.
pub fn write_schema_json<W: std::io::Write>(mut out: W) -> std::io::Result<()> {
    let s = schema_json();
    out.write_all(s.as_bytes())?;
    out.write_all(b"\n")
}
