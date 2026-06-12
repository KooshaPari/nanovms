use nvms_sdk::{NvmsClient, Result};

#[tokio::main]
async fn main() -> Result<()> {
    let client = NvmsClient::new("http://127.0.0.1:8080").await?;
    let _ = client;
    Ok(())
}
