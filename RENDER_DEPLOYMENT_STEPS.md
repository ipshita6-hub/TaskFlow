# 🚀 Render Deployment - Step by Step with Screenshots

## Overview
We'll deploy TaskFlow in 3 parts:
1. ✅ Create PostgreSQL Database (2 min)
2. ✅ Deploy Web Service (3 min)
3. ✅ Test the API (1 min)

**Total time: ~6 minutes**

---

## Part 1: Create Render Account

### Step 1.1: Go to Render
- Open https://render.com in your browser
- Click **"Sign up with GitHub"** button
- Click **"Authorize render-oss"** when prompted
- ✅ Account created!

---

## Part 2: Create PostgreSQL Database

### Step 2.1: Create New Database
1. In Render dashboard, click **+ New** (top right)
2. Select **PostgreSQL**
3. Fill in the form:
   ```
   Name:           taskflow-db
   Database:       taskflow
   User:           taskflow
   Region:         Oregon (or closest to you)
   PostgreSQL:     15
   Plan:           Free
   ```
4. Click **Create Database**
5. ⏳ Wait 2-3 minutes for database to initialize

### Step 2.2: Get Connection String
1. Once database shows "Available", open it
2. Find the **Internal Database URL** (starts with `postgres://`)
3. Copy it - **you'll need this in 5 minutes**
   - Format: `postgres://taskflow:PASSWORD@db-host:5432/taskflow`

**✅ Database is ready!**

---

## Part 3: Deploy Web Service

### Step 3.1: Create Web Service
1. Click **+ New** in Render dashboard
2. Select **Web Service**
3. Click **Connect a repository**
4. Find and select `TaskFlow` (yours)
5. Click **Connect**

### Step 3.2: Configure Service
Fill in the form:
```
Name:               taskflow-api
Root Directory:     (leave empty)
Runtime:            Docker
Region:             Oregon (match database)
Branch:             main
Plan:               Free
```

### Step 3.3: Add Environment Variables
1. Scroll to **Environment Variables**
2. Click **Add Environment Variable** for each:

**First Variable:**
- Key: `DATABASE_URL`
- Value: (paste the PostgreSQL connection string from Step 2.2)

**Second Variable:**
- Key: `JWT_SECRET`
- Value: (generate random: go to https://generate-random.org/ and generate a 32+ character string, OR use this command in terminal:)
  ```bash
  openssl rand -base64 32
  ```
  Then paste the result

**Leave these at defaults** (already in render.yaml):
- `JWT_EXPIRY_HOURS` = `24`
- `WORKER_CONCURRENCY` = `5`
- `SCHEDULER_TICK_MS` = `5000`
- `HEARTBEAT_INTERVAL_S` = `30`
- `STALE_JOB_THRESHOLD_S` = `90`
- `LOG_RETENTION_DAYS` = `30`
- `SERVER_PORT` = `8080`

### Step 3.4: Deploy!
1. Click **Create Web Service**
2. ⏳ Render will:
   - Build the Docker image (~3 min)
   - Push to registry
   - Deploy service
3. Watch the logs scroll by
4. Once you see **"Running"** - deployment complete! 🎉

---

## Part 4: Verify Deployment

### Step 4.1: Get Your API URL
Once deployment shows "Live":
- Find the URL at top: `https://taskflow-api.onrender.com`
- This is your API base URL

### Step 4.2: Test the API

**Test 1: Check Health (API Docs)**
```bash
curl https://taskflow-api.onrender.com/api/docs
```
Expected: HTML page loads (OpenAPI documentation)

**Test 2: Register a User**
```bash
curl -X POST https://taskflow-api.onrender.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```
Expected response:
```json
{
  "id": "uuid-here",
  "email": "test@example.com",
  "created_at": "2026-07-08T12:00:00Z"
}
```

**Test 3: Login**
```bash
curl -X POST https://taskflow-api.onrender.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```
Expected response:
```json
{
  "token": "eyJ...",
  "expires_at": "2026-07-09T12:00:00Z"
}
```

**Copy the token** - you'll use it for the next test

**Test 4: Create a Task**
```bash
curl -X POST https://taskflow-api.onrender.com/api/v1/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "name": "Test Task",
    "description": "My first task",
    "task_type": "noop",
    "schedule_type": "one_time",
    "scheduled_at": "2026-07-08T13:00:00Z",
    "retry_policy": {
      "max_attempts": 3,
      "backoff_seconds": 60
    }
  }'
```
Expected: Task created successfully

**Test 5: List Tasks**
```bash
curl https://taskflow-api.onrender.com/api/v1/tasks \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```
Expected: See your task in the list

✅ **All tests pass? Deployment is successful!**

---

## Part 5: Important Notes

### ⚠️ Free Tier Limitations
- **CPU**: 0.5 cores
- **Memory**: 512 MB
- **Sleep**: Service sleeps after 15 mins of no traffic
  - First request after sleep takes 10-30 seconds (normal)
  - Workaround: Upgrade to Starter plan ($7/month) for always-on
- **Build time**: ~3-5 minutes per deployment

### ✅ What Works on Free Tier
- All API endpoints ✓
- Task creation and scheduling ✓
- Worker execution ✓
- Database persistence ✓
- Background scheduler ✓
- Task retries ✓
- Execution logging ✓

### 🔄 Auto-Deployment
- Every push to `main` branch triggers new deployment
- Old service keeps running until new one is ready
- Automatic rollback if deployment fails

### 📊 Monitor Your Deployment
1. Go to dashboard → taskflow-api service
2. **Logs** tab: see real-time logs
3. **Events** tab: deployment history
4. **Metrics** tab: CPU, memory usage

---

## Part 6: Troubleshooting

### Problem: "Service Crashed" or "Failed to deploy"
**Solution:**
1. Click the service → **Logs** tab
2. Look for error messages
3. Common issues:
   - Missing `DATABASE_URL` → re-add env var
   - `JWT_SECRET` empty → generate new one
   - Database not ready → wait and redeploy

### Problem: "Connection to database failed"
**Solution:**
1. Verify `DATABASE_URL` is correct (copy it fresh from database dashboard)
2. Ensure you used the **Internal Database URL** (not external)
3. Check database status shows "Available"

### Problem: "Authentication failed"
**Solution:**
- Make sure `JWT_SECRET` is set
- Regenerate if unsure: `openssl rand -base64 32`

### Problem: "Service is sleeping"
**Solution:**
- Normal on free tier after 15 mins idle
- First request wakes it (10-30 sec delay)
- Upgrade plan to prevent this

---

## Part 7: Next Steps

### ✅ You've Deployed!
Celebrate! 🎉 Your API is live!

### 📝 What to do next:
1. **Test more endpoints** using your token
2. **Create recurring tasks** with cron expressions
3. **View execution logs** at `/api/v1/tasks/{id}/logs`
4. **Monitor workers** at `/api/v1/workers`

### 💰 Ready for Production?
Upgrade to **Starter plan** ($7/month):
- Always-on (no sleep)
- 1 CPU, 1GB RAM
- Perfect for small/medium workloads

Or use one of the alternatives:
- Railway.app
- Fly.io
- DigitalOcean
- AWS/GCP/Azure

---

## Part 8: Quick Reference URLs

After deployment, bookmark these:

| Purpose | URL |
|---------|-----|
| API Docs | `https://taskflow-api.onrender.com/api/docs` |
| Register | `POST https://taskflow-api.onrender.com/api/v1/auth/register` |
| Login | `POST https://taskflow-api.onrender.com/api/v1/auth/login` |
| Create Task | `POST https://taskflow-api.onrender.com/api/v1/tasks` |
| List Tasks | `GET https://taskflow-api.onrender.com/api/v1/tasks` |
| Task Details | `GET https://taskflow-api.onrender.com/api/v1/tasks/{id}` |
| Task Logs | `GET https://taskflow-api.onrender.com/api/v1/tasks/{id}/logs` |
| Failed Jobs | `GET https://taskflow-api.onrender.com/api/v1/jobs/dlq` |
| Workers | `GET https://taskflow-api.onrender.com/api/v1/workers` |

---

## Need Help?

| Problem | Where to Find Help |
|---------|-------------------|
| Render dashboard issues | https://render.com/docs |
| API doesn't work | Check logs in Render dashboard → service → Logs |
| Cron syntax | https://crontab.guru |
| API documentation | https://taskflow-api.onrender.com/api/docs |
| GitHub repo | https://github.com/ipshita6-hub/TaskFlow |

---

## Success Checklist

- [ ] Render account created
- [ ] PostgreSQL database created and "Available"
- [ ] Web service deployed and "Live"
- [ ] Environment variables set correctly
- [ ] Can access `https://taskflow-api.onrender.com/api/docs`
- [ ] Can register user (Test 2)
- [ ] Can login (Test 3)
- [ ] Can create task (Test 4)
- [ ] Can list tasks (Test 5)

✅ **All checked? You're done!**

🎉 **TaskFlow is live and ready to use!**
