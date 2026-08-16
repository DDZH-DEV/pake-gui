@echo off
cd /d "%~dp0"

echo Building Pake GUI desktop client...
go build -ldflags="-H windowsgui -s -w" -o PakeGUI.exe .
if errorlevel 1 (
  echo Build failed.
  exit /b 1
)

echo.
echo Done: %cd%\PakeGUI.exe
echo Double-click PakeGUI.exe to launch.
