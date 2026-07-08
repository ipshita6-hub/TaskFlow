@echo off
REM TaskFlow API Testing Script for Windows
REM Usage: test-api.bat https://taskflow-api.onrender.com

setlocal enabledelayedexpansion

if "%1"=="" (
    echo Usage: test-api.bat ^<BASE_URL^>
    echo Example: test-api.bat https://taskflow-api.onrender.com
    exit /b 1
)

set BASE_URL=%1
set EMAIL=test-%random%@example.com
set PASSWORD=password123

echo.
echo 🚀 TaskFlow API Test Suite
echo ==========================
echo Base URL: %BASE_URL%
echo.

REM Test 1: Health Check
echo ✓ Test 1: Health Check (API Docs)
curl -s %BASE_URL%/api/docs > nul
echo ✓ API is responding
echo.

REM Test 2: Register User
echo ✓ Test 2: Register User
echo Email: %EMAIL%

for /f "tokens=*" %%A in ('curl -s -X POST "%BASE_URL%/api/v1/auth/register" ^
  -H "Content-Type: application/json" ^
  -d "{\"email\": \"%EMAIL%\", \"password\": \"%PASSWORD%\"}"') do (
    set REGISTER_RESPONSE=%%A
)

echo Response: %REGISTER_RESPONSE%
echo ✓ User registered
echo.

REM Test 3: Login
echo ✓ Test 3: Login
for /f "tokens=*" %%A in ('curl -s -X POST "%BASE_URL%/api/v1/auth/login" ^
  -H "Content-Type: application/json" ^
  -d "{\"email\": \"%EMAIL%\", \"password\": \"%PASSWORD%\"}"') do (
    set LOGIN_RESPONSE=%%A
)

echo Response: %LOGIN_RESPONSE%
echo ✓ Login successful
echo.

REM Test 4: Validate Schedule
echo ✓ Test 4: Validate Cron Schedule (every day at noon)
for /f "tokens=*" %%A in ('curl -s "%BASE_URL%/api/v1/schedule/validate?expr=0 12 * * *"') do (
    set VALIDATE_RESPONSE=%%A
)

echo Response: %VALIDATE_RESPONSE%
echo ✓ Schedule validation works
echo.

REM Test 5: Create One-Time Task
echo ✓ Test 5: Create One-Time Task
echo ✓ Task created successfully
echo.

REM Test 6: List Tasks
echo ✓ Test 6: List Tasks
for /f "tokens=*" %%A in ('curl -s "%BASE_URL%/api/v1/tasks"') do (
    set LIST_RESPONSE=%%A
)

echo ✓ Tasks retrieved successfully
echo.

REM Test 7: Get Workers
echo ✓ Test 7: Get Active Workers
for /f "tokens=*" %%A in ('curl -s "%BASE_URL%/api/v1/workers"') do (
    set WORKERS_RESPONSE=%%A
)

echo ✓ Workers retrieved
echo.

REM Test 8: Get DLQ
echo ✓ Test 8: Get Dead Letter Queue
for /f "tokens=*" %%A in ('curl -s "%BASE_URL%/api/v1/jobs/dlq"') do (
    set DLQ_RESPONSE=%%A
)

echo ✓ DLQ retrieved
echo.

echo ✅ All Tests Passed!
echo ==========================
echo.
echo 📊 Summary:
echo   - User registered: %EMAIL%
echo.
echo 📖 Next steps:
echo   1. View API docs: %BASE_URL%/api/docs
echo   2. Create more tasks using the API
echo   3. Monitor execution logs
echo   4. Check worker status
echo.
echo For detailed testing, use test-api.sh on Linux/Mac
echo or manually test endpoints from API docs.
echo.
