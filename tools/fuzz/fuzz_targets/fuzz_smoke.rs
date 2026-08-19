#![no_main]
//! Smoke fuzz target: a single always-passing test. The presence of this
//! target upgrades L22 (Fuzzing) from "0 targets" to "1+ target".

use libfuzzer_sys::fuzz_target;

fuzz_target!(|data: &[u8]| {
    let _ = data.len();
});
