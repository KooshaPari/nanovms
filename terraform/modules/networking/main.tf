variable "environment" { type = string }
variable "region" { type = string }
output "vpc_id" { value = "vpc-${var.environment}-${var.region}" }
