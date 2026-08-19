variable "environment" { type = string }
variable "region" { type = string }
output "endpoint" { value = "https://k8s.${var.environment}.${var.region}.example.com" }
