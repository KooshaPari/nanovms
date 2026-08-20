output "cluster_name" {
  description = "The name of the EKS cluster"
  value       = aws_eks_cluster.main.name
}

output "cluster_endpoint" {
  description = "The endpoint for the EKS cluster"
  value       = aws_eks_cluster.main.endpoint
}

output "node_group_name" {
  description = "The name of the EKS node group"
  value       = aws_eks_node_group.sandbox.node_group_name
}

output "vpc_id" {
  description = "The ID of the VPC"
  value       = aws_vpc.main.id
}

output "nlb_dns_name" {
  description = "Placeholder for NLB DNS name if needed"
  value       = "placeholder-nlb-dns"
}
