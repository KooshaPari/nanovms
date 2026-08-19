// SPDX-License-Identifier: MIT OR Apache-2.0
//! NanoVMS Rust SDK
//!
//! Provides async client bindings and domain traits for interacting with
//! NanoVMS runtimes (WASM, gVisor, Firecracker) via REST or gRPC.

pub mod client;
pub mod config;
pub mod error;
pub mod models;
pub mod syscalls;

pub use client::NvmsClient;
pub use config::NvmsConfig;
pub use error::{NvmsError, Result};

use async_trait::async_trait;
use std::fmt::Debug;

/// A source of audio stream data for VM or sandbox contexts.
#[async_trait]
pub trait AudioSource: Send + Sync + Debug {
    /// Return the next chunk of audio samples.
    ///
    /// Returns `None` when the stream has ended.
    async fn next_chunk(&mut self) -> Result<Option<Vec<f32>>>;

    /// Sample rate in Hz (e.g., 44100, 48000).
    fn sample_rate(&self) -> u32;

    /// Number of audio channels (1 = mono, 2 = stereo).
    fn channels(&self) -> u16;

    /// Format identifier for the audio stream.
    fn format(&self) -> &str;
}

/// Renders audio, video, or telemetry output from a NanoVMS workload.
#[async_trait]
pub trait Renderer: Send + Sync + Debug {
    /// Accept a frame or buffer for rendering.
    ///
    /// The `frame` payload is opaque and should be interpreted based on the
    /// renderer variant (audio, video, or metrics).
    async fn render(&mut self, frame: RenderFrame) -> Result<()>;

    /// Signal that the stream has ended and resources should be released.
    async fn close(&mut self) -> Result<()>;

    /// Human-readable name of the renderer backend.
    fn name(&self) -> &str;
}

/// A single frame of render data passed to a [`Renderer`].
#[derive(Debug, Clone)]
pub struct RenderFrame {
    pub timestamp_ms: u64,
    pub data: FrameData,
    pub metadata: serde_json::Value,
}

/// Discriminated payload for render frames.
#[derive(Debug, Clone)]
pub enum FrameData {
    Audio { samples: Vec<f32> },
    Video { width: u32, height: u32, raw: Vec<u8> },
    Metrics { key: String, value: f64 },
    Text { payload: String },
}

/// Default no-op renderer for testing and placeholders.
#[derive(Debug)]
pub struct NullRenderer;

#[async_trait]
impl Renderer for NullRenderer {
    async fn render(&mut self, _frame: RenderFrame) -> Result<()> {
        Ok(())
    }

    async fn close(&mut self) -> Result<()> {
        Ok(())
    }

    fn name(&self) -> &str {
        "null"
    }
}

/// Default no-op audio source for testing and placeholders.
#[derive(Debug)]
pub struct NullAudioSource;

#[async_trait]
impl AudioSource for NullAudioSource {
    async fn next_chunk(&mut self) -> Result<Option<Vec<f32>>> {
        Ok(None)
    }

    fn sample_rate(&self) -> u32 {
        44100
    }

    fn channels(&self) -> u16 {
        2
    }

    fn format(&self) -> &str {
        "f32le"
    }
}
