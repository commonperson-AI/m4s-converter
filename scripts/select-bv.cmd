@echo off
chcp 65001 >nul
setlocal
cd /d "%~dp0"
set "CONVERTER=%~dp0m4s-converter-bv.exe"
if not exist "%CONVERTER%" (
  echo Missing m4s-converter-bv.exe. Extract the whole ZIP first.
  pause
  exit /b 1
)
"%CONVERTER%" -c "D:\BaiduNetdiskDownload\bilibili" --select -a
pause
