#!/bin/bash

# Configuration
SERVER="192.168.200.252"
USER="mobile"
PASS="bismillahAdmin" # Provided by user
# For automated password entry, consider using sshpass:
# sshpass -p "$PASS" ssh ...
# sshpass -p "$PASS" rsync ...
TARGET_DIR="/home/mobile/ziyad-messenger"

echo "🚀 Preparing deployment to $SERVER..."

# 1. Sync files to server
echo "📦 Syncing files (Enter password if prompted)..."
ssh $USER@$SERVER "mkdir -p $TARGET_DIR"

# Rsync project files, excluding large/unnecessary directories
rsync -avz --delete \
          --exclude '.git' \
          --exclude '*/node_modules' \
          --exclude 'frontend/admin-dashboard/.next' \
          --exclude 'data' \
          --exclude 'clients' \
          ./ $USER@$SERVER:$TARGET_DIR

# 2. Run remote commands
echo "🛠️  Building and starting services on $SERVER..."
ssh $USER@$SERVER << EOF
    cd $TARGET_DIR
    # Ensure env is correct for the internal local network IP
    sed -i "s/60.60.111.97/$SERVER/g" docker-compose.yml || true
    sed -i "s/api-dev.ziyadbooks.com/$SERVER/g" docker-compose.yml || true
    
    docker compose down
    docker compose build
    docker compose up -d
    docker compose ps
EOF

echo "✅ Deployment complete!"
