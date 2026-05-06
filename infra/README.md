# ECS Nitro Enclaves Terraform

Low-cost ECS on EC2 stack for Nitro Enclaves experiments.

This uses one Graviton ECS host because Nitro Enclaves cannot run on Fargate or
free-tier T-family EC2 instances. The default `c6g.large` in `us-east-1d` is a
low-cost Spot setting that also has the cheapest checked on-demand fallback.

## Cost Shape

- `c6gd.large` Spot in `us-east-1b`: about `$0.0221/hr` when checked, but Spot
  capacity was unavailable during deployment testing.
- `c6g.large` Spot in `us-east-1d`: about `$0.0226/hr` when checked.
- `c6g.large` Linux on-demand in `us-east-1`: `$0.068/hr`, which was the
  cheapest non-interruptible supported option checked.
- Spot is enabled by default with `use_spot = true` for the cheapest
  experiments.
- Nitro Enclaves has no extra service charge; the EC2 instance, EBS volume, and
  normal AWS services still cost money.
- No NAT gateway, no load balancer, and no inbound SSH are created.
- The 30 GiB encrypted gp3 root volume is deleted with the instance. The
  ECS-optimized Amazon Linux 2023 AMI snapshot requires at least 30 GiB.

Spot prices and capacity move. Re-check Spot before long runs. Spot is cheap
but interruptible; if the Spot instance is interrupted, the ECS host and any
running enclave terminate.

## What This Creates

- VPC with one public subnet and internet gateway.
- ECS cluster.
- One EC2 Auto Scaling Group with desired capacity `1`.
- ECS capacity provider backed by that Auto Scaling Group.
- Launch template with Nitro Enclaves enabled.
- ECS-optimized Amazon Linux 2023 ARM64 AMI from SSM.
- IAM instance profile for ECS and SSM Session Manager.
- Host bootstrap that installs Nitro Enclaves CLI packages and preallocates
  `512 MiB` plus `1 vCPU` for enclaves.

The `enclave_memory_mib` value can be reduced only if the enclave image can run
with less memory. It does not make AWS cheaper because EC2 bills for the whole
parent instance, not the enclave allocation.

## Usage

```sh
terraform init
terraform fmt
terraform validate
terraform plan
terraform apply
```

Use on-demand capacity instead of Spot:

```sh
terraform apply -var='use_spot=false' -var='instance_type=c6g.large'
```

Destroy the stack when you are done:

```sh
terraform destroy
```
