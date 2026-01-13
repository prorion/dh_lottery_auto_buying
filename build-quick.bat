@echo off
chcp 65001 >nul
echo 🚀 빠른 빌드 (현재 디렉토리)
echo.

echo [1/2] Windows 빌드...
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o dhlottery.exe .
if %ERRORLEVEL% neq 0 (
    echo ❌ 빌드 실패
    pause
    exit /b 1
)
echo ✅ dhlottery.exe 빌드 완료
echo.

echo [2/2] Linux/Ubuntu 빌드...
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o dhlottery-linux .
if %ERRORLEVEL% neq 0 (
    echo ❌ 빌드 실패
    pause
    exit /b 1
)
echo ✅ dhlottery-linux 빌드 완료
echo.

echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
echo 🎉 빌드 완료!
echo   - Windows: dhlottery.exe
echo   - Linux  : dhlottery-linux
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
pause
