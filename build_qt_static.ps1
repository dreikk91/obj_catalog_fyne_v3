param(
    [string]$Output = "dist\obj_catalog_qt_static.exe",
    [string]$MsysRoot = "D:\msys64"
)

$ErrorActionPreference = "Stop"

$qtRoot = Join-Path $MsysRoot "ucrt64\qt6-static"
$ucrtBin = Join-Path $MsysRoot "ucrt64\bin"
$goRoot = Join-Path $MsysRoot "ucrt64\lib\go"
$pkgConfigPath = Join-Path $PSScriptRoot "qt-static\pkgconfig"
$goExe = Join-Path $ucrtBin "go.exe"
$objdumpExe = Join-Path $ucrtBin "objdump.exe"

$requiredPaths = @(
    $goExe,
    (Join-Path $ucrtBin "gcc.exe"),
    (Join-Path $ucrtBin "g++.exe"),
    (Join-Path $ucrtBin "pkg-config.exe"),
    (Join-Path $qtRoot "lib\libQt6Core.a"),
    (Join-Path $qtRoot "lib\libQt6Gui.a"),
    (Join-Path $qtRoot "lib\libQt6Widgets.a")
)

foreach ($path in $requiredPaths) {
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Required static-build dependency is missing: $path"
    }
}

$qtVersion = & (Join-Path $qtRoot "bin\qmake.exe") -query QT_VERSION
if ($LASTEXITCODE -ne 0 -or $qtVersion -ne "6.11.1") {
    throw "Expected MSYS2 Qt 6.11.1, found '$qtVersion'. Update qt-static/pkgconfig after changing Qt."
}

if ([System.IO.Path]::IsPathRooted($Output)) {
    $outputPath = [System.IO.Path]::GetFullPath($Output)
} else {
    $outputPath = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot $Output))
}
$outputDirectory = Split-Path -Parent $outputPath
New-Item -ItemType Directory -Force -Path $outputDirectory | Out-Null

$qt = $qtRoot.Replace("\", "/")
$staticLinkFlags = @(
    "-static",
    "-static-libgcc",
    "-static-libstdc++",
    "-L$qt/share/qt6/plugins/styles",
    "-lqmodernwindowsstyle",
    "$qt/lib/objects-Release/QWindowsIntegrationPlugin_resources_1/.qt/rcc/qrc_openglblacklists_init.cpp.obj",
    "$qt/lib/objects-Release/QWindowsIntegrationPlugin_resources_2/.qt/rcc/qrc_cursors_init.cpp.obj",
    "-L$qt/share/qt6/plugins/platforms",
    "-lqwindows",
    "-limm32", "-loleaut32", "-lsetupapi", "-lwinspool", "-lwtsapi32",
    "-lshcore", "-lcomdlg32", "-ld3d9",
    "-L$qt/lib",
    "-lQt6OpenGL",
    "$qt/lib/objects-Release/Widgets_resources_1/.qt/rcc/qrc_qstyle_init.cpp.obj",
    "$qt/lib/objects-Release/Widgets_resources_2/.qt/rcc/qrc_qstyle1_init.cpp.obj",
    "$qt/lib/objects-Release/Widgets_resources_3/.qt/rcc/qrc_qstyle_fusion_init.cpp.obj",
    "$qt/lib/objects-Release/Widgets_resources_4/.qt/rcc/qrc_qmessagebox_init.cpp.obj",
    "-lQt6Widgets",
    "-ldwmapi",
    "$qt/lib/objects-Release/Gui_resources_1/.qt/rcc/qrc_qpdf_init.cpp.obj",
    "$qt/lib/objects-Release/Gui_resources_2/.qt/rcc/qrc_gui_shaders_init.cpp.obj",
    "-lQt6Gui",
    "-ld3d11", "-ldxgi", "-ldxguid", "-ld3d12", "-luxtheme",
    "-lpng", "-lpng16", "-latomic", "-lpcre2-8", "-lglib-2.0",
    "-lusp10", "-lgdi32", "-lshlwapi", "-lintl", "-lm",
    "-lgraphite2", "-lrpcrt4", "-lbz2", "-lharfbuzz", "-lfreetype",
    "-lbrotlidec", "-lbrotlicommon", "-ld2d1", "-ldwrite",
    "-lQt6Core",
    "-lz", "-lsynchronization", "-lmpr", "-luserenv", "-ladvapi32",
    "-lauthz", "-lkernel32", "-lnetapi32", "-lntdll", "-lole32",
    "-lruntimeobject", "-lshell32", "-luser32", "-luuid", "-lversion",
    "-lwinmm", "-lws2_32", "-lb2", "-lpcre2-16",
    "-lgraphite2", "-lrpcrt4", "-lusp10", "-lbz2"
)

$env:PATH = "$ucrtBin;$env:PATH"
$env:GOROOT = $goRoot
$env:CC = Join-Path $ucrtBin "gcc.exe"
$env:CXX = Join-Path $ucrtBin "g++.exe"
$env:CGO_ENABLED = "1"
$env:PKG_CONFIG_PATH = $pkgConfigPath
$env:PKG_CONFIG_DONT_DEFINE_PREFIX = "1"
$env:CGO_LDFLAGS = $staticLinkFlags -join " "

Write-Host "Building static Qt application with Qt $qtVersion..."
& $goExe build `
    -tags "qt,windowsqtstatic" `
    -trimpath `
    -buildvcs=false `
    -ldflags "-s -w -H=windowsgui" `
    -o $outputPath `
    ".\cmd\obj-catalog-qt"

if ($LASTEXITCODE -ne 0) {
    throw "Static Qt build failed with exit code $LASTEXITCODE."
}

Write-Host "Built: $outputPath"

if (Test-Path -LiteralPath $objdumpExe) {
    $nonSystemImports = & $objdumpExe -p $outputPath |
        Select-String "DLL Name:" |
        ForEach-Object { $_.Line.Trim() } |
        Where-Object { $_ -match "Qt6|libgcc|libstdc\+\+|libwinpthread" }

    if ($nonSystemImports) {
        throw "The output still imports toolchain DLLs: $($nonSystemImports -join ', ')"
    }

    Write-Host "Verified: no Qt or MinGW runtime DLL imports."
}
