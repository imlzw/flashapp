# FlashApp Linux Build Script
$ErrorActionPreference = "Stop"

Write-Host "正在准备构建 Linux 版本..." -ForegroundColor Cyan

# 设置交叉编译环境变量
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

# 设置缓存目录（参考 run.ps1）
$cacheDir = Join-Path $PSScriptRoot ".gocache"
if (!(Test-Path $cacheDir)) {
    New-Item -ItemType Directory -Force -Path $cacheDir | Out-Null
}
$env:GOCACHE = $cacheDir

$outputName = "flashapp_linux"
$sourcePath = "./src/cmd/flashapp"

Write-Host "正在构建: $outputName ..." -ForegroundColor Yellow

# 执行构建
& go build -o $outputName $sourcePath

if ($LASTEXITCODE -eq 0) {
    Write-Host "构建成功！" -ForegroundColor Green
    Write-Host "输出文件: $(Join-Path $PSScriptRoot $outputName)" -ForegroundColor Gray
} else {
    Write-Host "构建失败！" -ForegroundColor Red
    exit $LASTEXITCODE
}

# 恢复环境变量（可选，但在当前进程中执行时是个好习惯）
$env:GOOS = "windows"
$env:CGO_ENABLED = ""
