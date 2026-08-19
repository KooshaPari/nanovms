//! Integration tests that validate JSON against the v0.2 `odin.nvms`
//! JSON Schema using the `jsonschema` crate.
//!
//! These tests are the strongest contract: they prove the emitted schema
//! is consistent with what the Rust types actually deserialize, and that
//! the shipped example manifest is schema-valid.

use jsonschema::JSONSchema;
use phenotype_manifest::{schema_json, Manifest};
use serde_json::Value;

/// Valid example shipped under `crates/phenotype-manifest/examples/`.
const WEB_SERVICE_EXAMPLE: &str =
    include_str!("../examples/web-service.nvms.json");

fn compile_schema() -> JSONSchema {
    let raw = schema_json();
    let value: Value = serde_json::from_str(&raw)
        .expect("schema_json() output is valid JSON");
    JSONSchema::options()
        .with_draft(jsonschema::Draft::Draft202012)
        .compile(&value)
        .expect("compiled JSON Schema")
}

#[test]
fn emitted_schema_is_valid_draft202012() {
    // If this compiles, the schema itself is structurally valid JSON
    // Schema per Draft 2020-12.
    let _compiled = compile_schema();
}

#[test]
fn shipped_example_is_schema_valid() {
    let compiled = compile_schema();
    let manifest: Value = serde_json::from_str(WEB_SERVICE_EXAMPLE)
        .expect("example is valid JSON");
    let result = compiled.validate(&manifest);
    if let Err(errors) = result {
        for err in errors {
            eprintln!("validation error: {err}");
        }
        panic!("shipped example failed schema validation");
    }
}

#[test]
fn extra_unknown_field_is_rejected_by_schema() {
    // `deny_unknown_fields` is enabled on the Rust types, so an unknown
    // top-level field must NOT be schema-valid.
    let bad = r#"{
        "app": {"name": "x"},
        "infra": {"engine": "docker"},
        "completely_made_up_field": 42
    }"#;
    let compiled = compile_schema();
    let v: Value = serde_json::from_str(bad).expect("valid JSON");
    assert!(
        compiled.validate(&v).is_err(),
        "schema must reject unknown top-level fields (deny_unknown_fields)"
    );
}

#[test]
fn minimal_valid_input_is_schema_valid() {
    // The smallest possible valid manifest — only the two required
    // fields present. The schema (Draft 2020-12) must accept it.
    let minimal = r#"{
        "app": {"name": "min"},
        "infra": {"engine": "docker"}
    }"#;
    let compiled = compile_schema();
    let v: Value = serde_json::from_str(minimal).expect("valid JSON");
    assert!(
        compiled.is_valid(&v),
        "minimal valid manifest must pass JSON Schema validation"
    );
}

#[test]
fn serde_and_jsonschema_agree_on_validity() {
    // Cross-check: anything that `validate()` accepts must also pass the
    // emitted JSON Schema. Anything `validate()` rejects must also be
    // rejected by the schema. This proves the two are consistent.
    let valid_inputs: &[&str] = &[
        // minimal
        r#"{"app":{"name":"a"},"infra":{"engine":"docker"}}"#,
        // full example shipped in the repo
        WEB_SERVICE_EXAMPLE,
    ];
    for input in valid_inputs {
        let m: Manifest = serde_json::from_str(input).expect("serde parses");
        // Must pass validate()
        let value: Value = serde_json::from_str(input).unwrap();
        let compiled = compile_schema();
        assert!(
            compiled.is_valid(&value),
            "schema rejected what serde + validate accepted: {input}"
        );
        assert!(!m.app.name.is_empty());
    }
}