use serde::{Deserialize, Serialize};

/// A virtual machine record returned by the NanoVMS API.
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Vm {
    pub id: String,
    pub name: String,
    pub flavor: String,
    pub status: String,
    pub cpu: u32,
    pub memory: u64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub created_at: Option<String>,
}

/// A sandbox record returned by the NanoVMS API.
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Sandbox {
    pub id: String,
    pub name: String,
    pub tier: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub vm_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub created_at: Option<String>,
}

/// A network record returned by the NanoVMS API.
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Network {
    pub id: String,
    pub name: String,
    pub r#type: String,
    pub subnet: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub gateway: Option<String>,
    pub dhcp_enabled: bool,
}
