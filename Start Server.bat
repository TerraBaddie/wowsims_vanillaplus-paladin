@echo off
setlocal
cd /d "%~dp0"

echo ==========================================================
echo   WoWSims Classic - dev server
echo   Folder: %CD%
echo.
echo   CLOSE THIS WINDOW (or press Ctrl+C) TO STOP THE SERVER.
echo ==========================================================
echo.

where docker >nul 2>nul
if errorlevel 1 (
    echo [ERROR] Docker was not found. Start Docker Desktop and try again.
    echo.
    pause
    exit /b 1
)

echo Cleaning up any previous container...
docker rm -f wowsims >nul 2>nul

echo Opening browser in 90 seconds (first build takes a bit)...
start "" /min cmd /c "timeout /t 90 >nul & start "" http://localhost:8080/classic"

echo Starting server (auto-rebuilds UI, browser cache disabled)...
echo.
docker run --rm -it --name wowsims -p 8080:8080 -v "%CD%:/classic" -w /classic -e CHOKIDAR_USEPOLLING=true -e CHOKIDAR_INTERVAL=1000 wowsims-classic bash -c "make dist/classic/.dirstamp && (npx vite build -m development --watch > /tmp/vite.log 2>&1 &) && npx http-server 'dist/classic/..' -c-1 -p 8080"

echo.
echo Server stopped.
docker rm -f wowsims >nul 2>nul
timeout /t 3 >nul
endlocal
