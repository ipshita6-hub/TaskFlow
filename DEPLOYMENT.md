# Deployment Guide

This guide covers deploying TaskFlow to **Render** (recommended) or other platforms.

## Render Deployment (Recommended)

Render is the best choice for TaskFlow because it:
- ✅ Supports Go natively
- ✅ Allows long-running background processes (scheduler, workers)
- ✅ Includes managed PostgreSQL
- ✅ Auto-deploys from GitHub
- ✅ Generous free tier for testing

### Step 1: Push to GitHub (Already Done ✓)

Your repository is at: https://github.com/ipshita6-hub/TaskFlow

### Step 2: Create Render Account

1. Go to https://render.com
2. Click "Get Started" (free tier available)
3. Sign up with GitHub (recommended for auto-deploy)

### Step 3: Create PostgreSQL Database

1. In Render dashboard, click **+ New**
2. Select **PostgreSQL**
3. Configure:
   - **Name**: `taskflow-db`
   - **Database**: `taskflow`
   - **User**: `taskflow`
   - **Region**: Oregon (or closest to you)
   - **Plan**: Free
4. Click **Create Database**
5. Note the connection string (will be available after creation)

### Step 4: Deploy Web Service

1. Click **+ New**
2. Select **Web Service**
3. Connect your GitHub repository:
   - Click "Connect a repository"
   - Select `ipshita6-hub/TaskFlow`
4. Configure:
   - **Name**: `taskflow-api`
   - **Region**: Oregon
   - **Branch**: `main`
   - **Runtime**: Docker (will auto-detect Dockerfile)
   - **Plan**: Free
5. Set **Environment Variables**:
   - `DATABASE_URL`: Copy from PostgreSQL database creation
   - `JWT_SECRET`: Generate a random 32-char string or let Render auto-generate
   - Keep other defaults from `render.yaml`
6. Click **Create Web Service**

### Step 5: Wait for Deployment

- Render will build and deploy automatically
- Check deployment status in the dashboard
- Once "Live", your API is accessible at:
  ```
  https://taskflow-api.onrender.com/api/v1/
  https://taskflow-api.onrender.com/api/docs
  ```

### Step 6: Test Deployment

```bash
# Register a user
curl -X POST https://taskflow-api.onrender.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# View OpenAPI docs
open https://taskflow-api.onrender.com/api/docs
```

---

## Railway.app (Alternative)

Railway is similar to Render with slightly different UX:

### Quick Setup
1. Go to https://railway.app
2. Click "Start Project"
3. Select "Repo: Bring your own" or connect GitHub
4. Add PostgreSQL plugin
5. Deploy

---

## Fly.io (Alternative)

Fly.io requires Docker but is excellent for production:

### Prerequisites
```bash
brew install flyctl  # macOS
# or download from https://fly.io/docs/getting-started/installing-flyctl/
```

### Setup
```bash
# Login to Fly
flyctl auth login

# Create app in project directory
flyctl launch

# Deploy
flyctl deploy
```

---

## Self-Hosting on VPS

If you prefer full control, deploy on:
- **DigitalOcean Droplet** ($4-6/month)
- **Linode** ($5/month)
- **AWS EC2** (free tier available)
- **Hetzner** (very affordable)

### Basic VPS Deployment

```bash
# SSH into your VPS
ssh root@your-vps-ip

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Clone repository
git clone https://github.com/ipshita6-hub/TaskFlow.git
cd TaskFlow

# Create .env file
cat > .env << EOF
DATABASE_URL=postgres://user:password@db:5432/taskflow
JWT_SECRET=$(openssl rand -base64 32)
EOF

# Run with Docker Compose
docker-compose up -d

# Set up reverse proxy with Nginx
# (Details below)
```

### Nginx Reverse Proxy Setup

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Then enable with Let's Encrypt:
```bash
apt-get install certbot python3-certbot-nginx
certbot --nginx -d your-domain.com
```

---

## Troubleshooting

### Render: Database Connection Failed
- Check `DATABASE_URL` format matches PostgreSQL connection string
- Ensure database exists and service is running

### Render: Service Crashing
- Check logs: Dashboard → Service → Logs
- Verify all required env vars are set
- Check `JWT_SECRET` is not empty

### Port Issues
- TaskFlow listens on port 8080 (configured in `SERVER_PORT` env var)
- Render automatically maps this to https

### Cold Starts
- Free tier services sleep after 15 mins of inactivity
- First request after sleep may take 10-30 seconds
- Upgrade to paid plan for guaranteed uptime

---

## Production Checklist

- [ ] Set `JWT_SECRET` to strong random value (32+ chars)
- [ ] Set appropriate `WORKER_CONCURRENCY` (default 5)
- [ ] Configure `LOG_RETENTION_DAYS` (default 30)
- [ ] Set up regular database backups
- [ ] Enable HTTPS (automatic on Render)
- [ ] Monitor logs and set up alerts
- [ ] Test task execution end-to-end
- [ ] Set up monitoring/health checks

---

## Environment Variables Summary

| Variable | Example | Notes |
|----------|---------|-------|
| `DATABASE_URL` | `postgres://user:pass@host:5432/db` | Required |
| `JWT_SECRET` | `your-random-32-char-string` | Generate with `openssl rand -base64 32` |
| `JWT_EXPIRY_HOURS` | `24` | Token expiration |
| `WORKER_CONCURRENCY` | `5` | Number of concurrent task workers |
| `SCHEDULER_TICK_MS` | `5000` | Task scheduler polling interval |
| `HEARTBEAT_INTERVAL_S` | `30` | Worker health check interval |
| `STALE_JOB_THRESHOLD_S` | `90` | Mark job stale after N seconds |
| `LOG_RETENTION_DAYS` | `30` | Keep execution logs for N days |
| `SERVER_PORT` | `8080` | HTTP server port |

---

## Monitoring & Scaling

### Render Free Tier Limits
- 0.5 CPU, 512 MB RAM per service
- Auto-sleep after 15 mins inactivity
- Perfect for development/testing

### To Scale Up
1. Dashboard → Service → Plan → Upgrade
2. Recommended for production: Standard ($7/month)

### Logs & Metrics
- Render: Dashboard → Service → Logs
- Fly.io: `flyctl logs -a app-name`
- Railway: Dashboard → Logs

---

## Questions?

For issues with:
- **Render deployment**: https://render.com/docs
- **TaskFlow setup**: See README.md
- **Docker**: https://docs.docker.com
