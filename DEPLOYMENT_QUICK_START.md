# 🚀 Quick Deploy to Render (5 Minutes)

## Option A: Deploy to Render (Recommended)

### Prerequisites
- GitHub account (already have ✓)
- Render account (free at https://render.com)

### Step-by-Step

**1. Create Render Account**
- Go to https://render.com
- Click "Sign up with GitHub"
- Authorize TaskFlow repo

**2. Create PostgreSQL Database**
- Dashboard → **+ New** → **PostgreSQL**
- Name: `taskflow-db`
- Database: `taskflow`
- User: `taskflow`
- Region: Oregon (or closest)
- Plan: **Free**
- Click **Create Database**
- ⏳ Wait 2-3 minutes for database to create
- Copy the connection string (starts with `postgres://`)

**3. Deploy Web Service**
- Dashboard → **+ New** → **Web Service**
- Select repository: `TaskFlow`
- Configure:
  - **Name**: `taskflow-api`
  - **Runtime**: Docker (auto-detected)
  - **Region**: Oregon
  - **Plan**: Free
- Click **Connect**

**4. Set Environment Variables**
In the service configuration, add:
```
DATABASE_URL = <paste the PostgreSQL connection string>
JWT_SECRET = <generate at https://generate-random.org/ or use openssl rand -base64 32>
```
Leave others as default or from `render.yaml`

**5. Deploy**
- Click **Create Web Service**
- Render builds and deploys automatically (~3-5 minutes)
- Once "Live", note the URL: `https://taskflow-api.onrender.com`

**6. Test It**
```bash
# Check health
curl https://taskflow-api.onrender.com/api/docs

# Register user
curl -X POST https://taskflow-api.onrender.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123"}'
```

✅ **Done!** Your API is live!

---

## Option B: Deploy to Railway (Alternative)

1. Go to https://railway.app
2. Click "Start a New Project"
3. Select "Deploy from GitHub repo"
4. Choose `ipshita6-hub/TaskFlow`
5. Add PostgreSQL service
6. Set environment variables
7. Deploy

---

## Option C: Deploy to Fly.io (Alternative)

```bash
# Install Fly CLI
curl -L https://fly.io/install.sh | sh

# Login
flyctl auth login

# Deploy
cd TaskFlow
flyctl launch  # Follow prompts
flyctl deploy
```

---

## Accessing Your Deployment

Once deployed to Render at `https://taskflow-api.onrender.com`:

| Endpoint | URL |
|----------|-----|
| **API Base** | `https://taskflow-api.onrender.com/api/v1/` |
| **OpenAPI Docs** | `https://taskflow-api.onrender.com/api/docs` |
| **Register** | `POST /api/v1/auth/register` |
| **Login** | `POST /api/v1/auth/login` |
| **Create Task** | `POST /api/v1/tasks` |
| **List Tasks** | `GET /api/v1/tasks` |

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Database connection failed | Verify `DATABASE_URL` in env vars, check database is running |
| Service keeps crashing | Check logs in Render dashboard, verify all required env vars set |
| Cold start (first request slow) | Normal on free tier, upgrades prevent this |
| Service goes to sleep | Free tier services sleep after 15 mins, upgrade to prevent |

---

## Important Notes

⚠️ **Free Tier Limitations**
- 0.5 CPU, 512 MB RAM
- Auto-sleeps after 15 mins of no traffic
- Great for testing/development
- Upgrade to Standard ($7/month) for production

✅ **What Works on Free Tier**
- Full API functionality
- Task scheduling and execution
- Worker pools
- Database persistence
- All background processes

🔄 **Auto-Deploy from GitHub**
- Every push to `main` triggers deployment
- Takes 3-5 minutes
- Rollback available if needed

---

## Next Steps

1. ✅ Deploy to Render (completed above)
2. Register a user via the API
3. Create a task with a cron expression
4. Monitor execution via `/api/v1/tasks/{id}/logs`
5. Upgrade to paid plan when ready for production

---

## Support

- **Render docs**: https://render.com/docs
- **TaskFlow README**: See README.md for API details
- **Deployment guide**: See DEPLOYMENT.md for advanced options

🎉 **Congratulations on deploying TaskFlow!**
