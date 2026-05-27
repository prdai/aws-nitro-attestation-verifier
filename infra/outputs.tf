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

output "http_port" {
  description = "Public HTTP port allowed by the EC2 host security group."
  value       = var.http_port
}

output "http_url_hint" {
  description = "HTTP URL shape. Replace <ec2-public-ip> with the current ECS host public IPv4 address."
  value       = "http://<ec2-public-ip>:${var.http_port}/"
}

output "ec2_public_ips" {
  description = "Public IPv4 addresses for the running parent EC2 host instances."
  value       = data.aws_instances.ecs_hosts.public_ips
}

output "ec2_attestation_urls" {
  description = "Public attestation relay URL for each running ECS host instance. Use one as EC2_ATTESTATION_URL."
  value = [
    for public_ip in data.aws_instances.ecs_hosts.public_ips :
    "http://${public_ip}:${var.http_port}/attestation"
  ]
}

output "ssh_commands" {
  description = "SSH command templates for each running parent EC2 host. Replace the key path with your local private key path."
  value = [
    for public_ip in data.aws_instances.ecs_hosts.public_ips :
    "ssh -i ~/.ssh/${coalesce(var.ssh_key_name, "your-key")}.pem ec2-user@${public_ip}"
  ]
}
