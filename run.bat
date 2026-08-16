@echo off
cd /d "%~dp0"
if not exist PakeGUI.exe (
  call build.bat
  if errorlevel 1 exit /b 1
)
start "" "%~dp0PakeGUI.exe"
