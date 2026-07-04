./@echo off
setlocal EnableExtensions
chcp 65001 >nul

set "APP_PKG=.\cmd\obj-catalog-qt"
set "OUT=%~1"
if "%OUT%"=="" set "OUT=dist\obj_catalog_qt.exe"

for %%I in ("%OUT%") do set "OUT_DIR=%%~dpI"
if not "%OUT_DIR%"=="" if not exist "%OUT_DIR%" mkdir "%OUT_DIR%"

set "OUT_DIR_NO_SLASH=%OUT_DIR%"
if "%OUT_DIR_NO_SLASH:~-1%"=="\" set "OUT_DIR_NO_SLASH=%OUT_DIR_NO_SLASH:~0,-1%"

echo Building Qt release: %OUT%
echo.

echo Embedding icon and metadata...
go-winres simply ^
  --icon "icon.png" ^
  --arch amd64 ^
  --out "cmd\obj-catalog-qt\rsrc" ^
  --manifest gui ^
  --product-version "3.0.0.0" ^
  --file-version "3.0.0.0" ^
  --product-name "Object catalog - QT" ^
  --file-description "Catalog of objects - QT UI" ^
  --original-filename "obj_catalog_qt.exe" ^
  --copyright "2026"

if %ERRORLEVEL% NEQ 0 (
  echo.
  echo Warning: go-winres failed. Building without icon...
  echo.
)

go build ^
  -tags qt ^
  -trimpath ^
  -buildvcs=false ^
  -ldflags "-s -w -H=windowsgui" ^
  -o "%OUT%" ^
  %APP_PKG%

if %ERRORLEVEL% NEQ 0 (
  echo.
  echo Build failed.
  exit /b 1
)

echo.
echo Build completed: %OUT%
echo.

where windeployqt >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
  echo Warning: windeployqt not found in PATH.
  echo          Qt DLLs will not be deployed to "%OUT_DIR%".
  echo          To run on other PCs, please install Qt or add windeployqt to PATH.
) else (
  echo Deploying Qt and MinGW dependencies...
  windeployqt --release "%OUT%"
  if %ERRORLEVEL% NEQ 0 (
    echo Warning: Deployment with windeployqt failed.
  ) else (
    echo Deployment with windeployqt completed.
  )
  echo Resolving recursive dependencies...
  powershell -NoProfile -ExecutionPolicy Bypass -File "scripts\deploy_deps.ps1" -distDir "%OUT_DIR_NO_SLASH%"
  if %ERRORLEVEL% NEQ 0 (
    echo Warning: Recursive dependency check failed.
  ) else (
    echo All dependencies resolved successfully.
  )
)

echo.
echo NOTE: UPX compression disabled ??? CGO/Qt binaries are incompatible with UPX
echo       (causes 0xc0000142 at startup). The -s -w ldflags already strip
echo       debug info, reducing size from ~365MB to ~63MB.
exit /b 0
