#![no_main]
use libfuzzer_sys::fuzz_target;

fuzz_target!(|data: &[u8]| {
    // Placeholder: convert bytes to UTF-8 string to exercise basic parsing.
    let _ = std::str::from_utf8(data);
});
