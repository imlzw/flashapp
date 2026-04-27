param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Args
)

$cacheDir = Join-Path $PSScriptRoot ".gocache"
New-Item -ItemType Directory -Force -Path $cacheDir | Out-Null
$env:GOCACHE = $cacheDir

& go run ./src/cmd/flashapp @Args
