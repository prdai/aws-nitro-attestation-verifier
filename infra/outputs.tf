output "ecs_cluster_name" {
  description = "ECS cluster name."
  value       = aws_ecs_cluster.this.name
}

output "ecs_capacity_provider_name" {
  description = "ECS EC2 capacity provider name."
  value       = aws_ecs_capacity_provider.this.name
}

output "ecs_host_instance_type" {
  description = "EC2 instance type used for the ECS host."
  value       = var.instance_type
}

output "ecs_host_market" {
  description = "EC2 market option for the ECS host."
  value       = var.use_spot ? "spot" : "on-demand"
}

output "vpc_id" {
  description = "VPC ID."
  value       = aws_vpc.this.id
}

output "public_subnet_id" {
  description = "Public subnet ID used by the ECS host."
  value       = aws_subnet.public.id
}
