---
description: How to deploy the platform to a production/dev server via SSH
---

This workflow explains how to deploy the platform using Docker Compose on a remote server.

### Prerequisites

1. Docker and Docker Compose installed on the target server.
2. SSH access to the server (`mobile@api-dev.ziyadbooks.com`).
3. Your SSH public key added to the server's `~/.ssh/authorized_keys`.

### Steps

1. **Configure Environment**
   Ensure your `.env` or environment variables in `docker-compose.yml` are correct for the target server (e.g., ports, domains).

2. **Run Deployment Script**
   // turbo
   Execute the pre-configured deployment script:

   ```bash
   ./deploy/deploy.sh
   ```

3. **Verify Deployment**
   Check the status of the containers on the target server:

   ```bash
   ssh mobile@api-dev.ziyadbooks.com "cd ~/ziyad-messenger && docker-compose ps"
   ```

4. **Access the Dashboard**
   - **Admin Dashboard**: Open `http://api-dev.ziyadbooks.com:8091`
   - **Admin API**: `http://api-dev.ziyadbooks.com:8090`
   - **Auth API**: `http://api-dev.ziyadbooks.com:8086`

### Troubleshooting

- If the build fails on the server, check for memory limits (Next.js build can be intensive).
- Ensure port 3000, 8086, 8090, 8081, etc., are open in the server firewall.
