use thiserror::Error;

/// The unified error type for the NanoVMS Rust SDK.
#[derive(Debug, Error)]
pub enum NvmsError {
    /// Failed to initialize the HTTP client.
    #[error("client initialization failed: {0}")]
    ClientInit(String),

    /// The HTTP request failed.
    #[error("request failed: {0}")]
    RequestFailed(String),

    /// The server returned a non-success status code.
    #[error("HTTP {status}: {body}")]
    HttpStatus { status: u16, body: String },

    /// Failed to deserialize the response body.
    #[error("deserialization failed: {0}")]
    Deserialize(String),

    /// A validation or domain error.
    #[error("domain error: {0}")]
    Domain(String),
}

/// A convenient alias for `std::result::Result<T, NvmsError>`.
pub type Result<T> = std::result::Result<T, NvmsError>;
