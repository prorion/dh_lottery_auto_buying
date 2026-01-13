@echo off
chcp 65001 >nul

echo Linux/Ubuntu (amd64) 빌드 중...
powershell -Command "$env:CGO_ENABLED='0'; $env:GOOS='linux'; $env:GOARCH='amd64'; go build -ldflags='-s -w' -o dhlottery ."
if %ERRORLEVEL% neq 0 (
    echo ❌ 빌드 실패
    pause
    exit /b 1
)

echo.
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 🎉 빌드 완료!
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo.
pause
