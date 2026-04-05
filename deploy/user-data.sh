#!/bin/bash
set -euo pipefail

# ─────────────────────────────────────────────
# Cloud-init user-data script for invoice-bot EC2 instance
# Installs Docker, Docker Compose, and prepares the deployment directory.
# ─────────────────────────────────────────────

# Update system
dnf update -y

# Install Docker
dnf install -y docker

# Start and enable Docker
systemctl enable --now docker

# Add ec2-user to docker group (so we can run docker without sudo)
usermod -aG docker ec2-user

# Install Docker Compose plugin
mkdir -p /usr/local/lib/docker/cli-plugins
COMPOSE_VERSION="v2.29.1"
curl -SL "https://github.com/docker/compose/releases/download/$${COMPOSE_VERSION}/docker-compose-linux-x86_64" \
  -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

# Create application directory
mkdir -p /opt/invoice-bot
chown ec2-user:ec2-user /opt/invoice-bot

# Write the production docker-compose file
cat > /opt/invoice-bot/docker-compose.yml <<'COMPOSE'
services:
  invoice-bot:
    image: ${ecr_url}:latest
    restart: unless-stopped
    env_file:
      - .env
    environment:
      - TZ=Europe/London
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
COMPOSE

chown ec2-user:ec2-user /opt/invoice-bot/docker-compose.yml

# Write a helper script to login to ECR and pull the latest image
cat > /opt/invoice-bot/deploy.sh <<'DEPLOY'
#!/bin/bash
set -euo pipefail
aws ecr get-login-password --region ${aws_region} | docker login --username AWS --password-stdin ${ecr_url}
docker compose pull
docker compose up -d
DEPLOY

chmod +x /opt/invoice-bot/deploy.sh
chown ec2-user:ec2-user /opt/invoice-bot/deploy.sh

# Install the AWS CLI (needed for ECR login via instance profile)
dnf install -y aws-cli

echo "User-data setup complete. SSH in and create /opt/invoice-bot/.env to finish setup."
