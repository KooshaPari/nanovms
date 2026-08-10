//! Integration tests for `phenotype_manifest::validate`.
//!
//! TDD step 1 — these tests are written FIRST and must FAIL to compile
//! until `lib.rs` + `schema.rs` exist. They are the contract for the
//! v0.1 `odin.nvms` schema.

use phenotype_manifest::{validate, Manifest};

/// Valid example shipped under `crates/phenotype-manifest/examples/`.
const WEB_SERVICE_EXAMPLE: &str =
    include_str!("../examples/web-service.nvms.json");

#[test]
fn valid_example_parses_and_validates_ok() {
    // Ensure serde alone can deserialize the example (paranoia: catches
    // schema drift early).
    let _manifest: Manifest =
        serde_json::from_str(WEB_SERVICE_EXAMPLE).expect("serde parses example");
    let result = validate(WEB_SERVICE_EXAMPLE);
    assert!(
        result.is_ok(),
        "valid example must validate; got {:?}",
        result.err()
    );
    let m = result.expect("validate succeeded");
    assert_eq!(m.app.name, "web-service");
    assert_eq!(m.infra.engine, "docker");
}

#[test]
fn empty_app_name_fails_validate() {
    // app.name is required and cannot be empty per v0.1 contract.
    let bad = r#"{
        "app": {"name": "", "repo": null, "build": null, "runtime": null},
        "infra": {"engine": "docker", "resources": null}
    }"#;
    let result = validate(bad);
    assert!(result.is_err(), "empty app.name must return Err");
}

#[test]
fn invalid_json_returns_err() {
    let result = validate("not json at all");
    assert!(result.is_err(), "garbage input must return Err");
}

#[test]
fn missing_app_field_returns_err() {
    let bad = r#"{
        "infra": {"engine": "docker", "resources": null}
    }"#;
    let result = validate(bad);
    assert!(result.is_err(), "missing app must return Err");
}
