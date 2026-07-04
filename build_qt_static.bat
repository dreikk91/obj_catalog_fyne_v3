@echo off
setlocal EnableExtensions
chcp 65001 >nul

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0build_qt_static.ps1" %*
exit /b %ERRORLEVEL%
