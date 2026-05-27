#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

mode="${EC2_DEPLOY_MODE:-git}"
remote_dir="${EC2_REMOTE_DIR:-/home/ec2-user/aws-nitro-attestation-verifier}"
repo_url="${EC2_REPO_URL:-https://github.com/prdai/aws-nitro-attestation-verifier.git}"
repo_ref="${EC2_REPO_REF:-main}"
ssh_user="${EC2_SSH_USER:-ec2-user}"
ssh_key_path="${EC2_SSH_KEY_PATH:-$HOME/.ssh/id_prdai}"
host="${EC2_HOST:-}"

if [[ -z "$host" ]]; then
  host="$(terraform -chdir="$repo_root/infra" output -json ec2_public_ips | jq -r '.[0]')"
fi

if [[ -z "$host" || "$host" == "null" ]]; then
  echo "missing EC2 host: set EC2_HOST or run terraform apply first" >&2
  exit 1
fi

ssh_base=(
  ssh
  -o StrictHostKeyChecking=accept-new
  -i "$ssh_key_path"
  "$ssh_user@$host"
)

echo "ec2 host: $host"
echo "deploy mode: $mode"
echo "remote dir: $remote_dir"

"${ssh_base[@]}" "set -euxo pipefail; \
  for attempt in 1 2 3; do \
    sudo dnf install -y git make golang rsync docker aws-nitro-enclaves-cli aws-nitro-enclaves-cli-devel && break; \
    sudo dnf clean all || true; \
    sudo rm -rf /var/cache/dnf || true; \
    sleep 5; \
  done; \
  sudo systemctl enable --now docker nitro-enclaves-allocator.service"

case "$mode" in
  git)
    "${ssh_base[@]}" "set -euxo pipefail; if [ -d '$remote_dir/.git' ]; then cd '$remote_dir' && git fetch origin '$repo_ref' && git reset --hard 'origin/$repo_ref'; else rm -rf '$remote_dir' && git clone --branch '$repo_ref' '$repo_url' '$remote_dir'; fi"
    ;;
  sync)
    "${ssh_base[@]}" "mkdir -p '$remote_dir'"
    rsync -az --delete \
      --exclude '.git/' \
      --exclude '.terraform/' \
      --exclude '.terraform.lock.hcl' \
      --exclude '.cache/' \
      --exclude 'terraform.tfstate*' \
      --exclude 'nitro-attestation-enclave.eif' \
      -e "ssh -o StrictHostKeyChecking=accept-new -i $ssh_key_path" \
      "$repo_root/" "$ssh_user@$host:$remote_dir/"
    ;;
  *)
    echo "invalid EC2_DEPLOY_MODE=$mode; expected git or sync" >&2
    exit 1
    ;;
esac

"${ssh_base[@]}" "set -euxo pipefail; cd '$remote_dir'; \
  if [ -f ec2-relay.pid ]; then kill \$(cat ec2-relay.pid) 2>/dev/null || true; rm -f ec2-relay.pid; fi; \
  sudo nitro-cli terminate-enclave --enclave-name nitro-attestation-enclave 2>/dev/null || true; \
  sudo make enclave-docker-build; \
  sudo make enclave-eif-build; \
  sudo make enclave-run; \
  mkdir -p bin; \
  cd ec2; \
  env GOTOOLCHAIN=auto GOSUMDB=sum.golang.org GOPROXY=https://proxy.golang.org,direct go build -o ../bin/ec2-relay ./cmd/server; \
  nohup env ENCLAVE_CID=16 ENCLAVE_PORT=5000 HTTP_ADDR=:8080 ../bin/ec2-relay > ../ec2-relay.log 2>&1 & echo \$! > ../ec2-relay.pid; \
  cd ..; \
  for attempt in \$(seq 1 60); do \
    if curl -fsS http://127.0.0.1:8080/healthz; then \
      break; \
    fi; \
    sleep 1; \
  done; \
  curl -fsS http://127.0.0.1:8080/healthz; \
  cat ec2-relay.pid; \
  tail -n 40 ec2-relay.log"
