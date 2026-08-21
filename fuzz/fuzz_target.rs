#![no_main]
use arbitrary::Arbitrary;
libfuzzer_sys::fuzz_target!(|data: &[u8]| { let _ = data; });
