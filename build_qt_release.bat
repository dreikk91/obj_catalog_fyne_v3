@echo off
setlocal EnableExtensions

set "APP_PKG=.\cmd\obj-catalog-qt"
set "OUT=%~1"
if "%OUT%"=="" set "OUT=dist\obj_catalog_qt.exe"

for %%I in ("%OUT%") do set "OUT_DIR=%%~dpI"
if not "%OUT_DIR%"=="" if not exist "%OUT_DIR%" mkdir "%OUT_DIR%"

echo Building Qt release: %OUT%
echo.

go build ^
  -tags qt ^
  -trimpath ^
  -buildvcs=false ^
  -ldflags "-s -w -H=windowsgui" ^
  -o "%OUT%" ^
  %APP_PKG%

if errorlevel 1 (
  echo.
  echo Build failed.
  exit /b 1
)

echo.
echo Build completed.

where upx >nul 2>nul
if errorlevel 1 (
  echo UPX not found in PATH. Skipping compression.
  echo Output: %OUT%
  exit /b 0
)

echo.
echo Compressing with UPX...
upx --best --lzma "%OUT%"

if errorlevel 1 (
  echo.
  echo UPX compression failed. Uncompressed executable is still available:
  echo %OUT%
  exit /b 1
)

echo.
echo Done: %OUT%
exit /b 0
