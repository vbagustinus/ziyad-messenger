#!/bin/bash
set -euo pipefail

# Configuration
SERVER="192.168.200.252"
USER="mobile"
PASS="bismillahAdmin" # Provided by user
# For automated password entry, consider using sshpass:
# sshpass -p "$PASS" ssh ...
# sshpass -p "$PASS" rsync ...
TARGET_DIR="/home/mobile/ziyad-messenger"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "🚀 Preparing deployment to $SERVER..."

# 1. Sync files to server
echo "📦 Syncing files (Enter password if prompted)..."
ssh $USER@$SERVER "mkdir -p $TARGET_DIR"

# Rsync project files, excluding large/unnecessary directories
rsync -avz --delete \
          --exclude '.git' \
          --exclude '*/node_modules' \
          --exclude 'frontend/admin-dashboard/.next' \
          --exclude 'backend/deploy/data' \
          --exclude 'clients' \
          "$REPO_ROOT"/ $USER@$SERVER:$TARGET_DIR

# 2. Run remote commands
echo "🛠️  Building and starting services on $SERVER..."
ssh $USER@$SERVER << EOF
    cd $TARGET_DIR/backend
    # Ensure env is correct for the internal local network IP
    sed -i "s/60.60.111.97/$SERVER/g" deploy/docker-compose.yml || true
    sed -i "s/api-dev.ziyadbooks.com/$SERVER/g" deploy/docker-compose.yml || true
    
    docker compose -f deploy/docker-compose.yml down
    docker compose -f deploy/docker-compose.yml build
    docker compose -f deploy/docker-compose.yml up -d
    docker compose -f deploy/docker-compose.yml ps
EOF

echo "✅ Deployment complete!"
