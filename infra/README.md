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
- No NAT gateway or load balancer is created.
- Inbound SSH is disabled by default. Set `ssh_key_name` and
  `allowed_ssh_cidr_blocks` to clone/build the repo directly on the parent EC2
  host.
- Public HTTP ingress is allowed on port `8080` by default for the EC2-side
  request receiver experiment.
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
- Security group ingress for the EC2 HTTP server port.
- Optional security group ingress for SSH when `allowed_ssh_cidr_blocks` is
  configured.

The Terraform stack prepares the parent EC2 host. The intended current workflow
is to SSH into that host, clone this repository there, then use the root
`Makefile` enclave targets on the EC2 host to build the Docker image, convert it
to an EIF, start the enclave, and run the parent HTTP relay. Building the EIF on
the parent host is intentional because `nitro-cli` needs the Nitro-capable EC2
environment.

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

Get the public attestation relay URL after apply:

```sh
terraform output -json ec2_attestation_urls
```

Use on-demand capacity instead of Spot:

```sh
terraform apply -var='use_spot=false' -var='instance_type=c6g.large'
```

Enable SSH for the parent EC2 host with an existing EC2 key pair:

```sh
MY_IP=$(curl -fsSL https://checkip.amazonaws.com | tr -d '\n')
terraform apply \
  -var='ssh_key_name=your-ec2-key-pair-name' \
  -var="allowed_ssh_cidr_blocks=[\"${MY_IP}/32\"]"
```

After apply, print the SSH command and clone/build on the host:

```sh
terraform output -json ssh_commands | jq -r '.[]'
make ec2-app-deploy
```

`make ec2-app-deploy` clones the pushed `main` branch on the parent EC2 host,
builds the enclave image, converts it to an EIF, starts the enclave, and starts
the HTTP relay. For uncommitted local experiments, use
`make ec2-app-sync-deploy` to rsync the current working tree before running the
same remote build/start steps.

The manual equivalent is:

```sh
ssh -i ~/.ssh/your-ec2-key-pair-name.pem ec2-user@EC2_PUBLIC_IP
git clone https://github.com/prdai/aws-nitro-attestation-verifier.git
cd aws-nitro-attestation-verifier
make enclave-docker-build
make enclave-eif-build
make enclave-run
make ec2-server-run
```

Destroy the stack when you are done:

```sh
terraform destroy
```
