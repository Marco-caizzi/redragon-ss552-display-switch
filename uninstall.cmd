@echo off
rem Double-click launcher: removes the Display Switch plugin from StreamDock.
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0uninstall.ps1" %*
echo.
pause
