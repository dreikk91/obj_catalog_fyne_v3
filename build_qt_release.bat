@echo off
setlocal EnableExtensions

set "APP_PKG=.\cmd\obj-catalog-qt"
set "OUT=%~1"
if "%OUT%"=="" set "OUT=dist\obj_catalog_qt.exe"

for %%I in ("%OUT%") do set "OUT_DIR=%%~dpI"
if not "%OUT_DIR%"=="" if not exist "%OUT_DIR%" mkdir "%OUT_DIR%"

echo Building Qt release: %OUT%
echo.

echo Embedding icon and metadata...
go-winres simply ^
  --icon "icon.png" ^
  --arch amd64 ^
  --out "cmd\obj-catalog-qt\rsrc" ^
  --manifest gui ^
  --product-name "АРМ Пожежної Безпеки" ^
  --file-description "Каталог об'єктів - Qt UI" ^
  --original-filename "obj_catalog_qt.exe" ^
  --copyright "2024-2026"

if errorlevel 1 (
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

if errorlevel 1 (
  echo.
  echo Build failed.
  exit /b 1
)

echo.
echo Build completed: %OUT%
echo.
echo NOTE: UPX compression disabled — CGO/Qt binaries are incompatible with UPX
echo       (causes 0xc0000142 at startup). The -s -w ldflags already strip
echo       debug info, reducing size from ~365MB to ~63MB.
exit /b 0
