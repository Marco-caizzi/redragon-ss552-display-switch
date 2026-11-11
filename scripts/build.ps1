# Builds displayswitch.exe and the release zip.
# Usage (from the repository root, with Go installed): .\scripts\build.ps1
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$pluginDir = Join-Path $root "com.marco.displayswitch.sdPlugin"
$version = (Get-Content (Join-Path $pluginDir "manifest.json") -Raw | ConvertFrom-Json).Version

Push-Location $root
try {
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "tests failed" }

    $env:GOOS = "windows"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"
    go build -trimpath -ldflags "-H windowsgui -s -w" -o (Join-Path $pluginDir "displayswitch.exe") ./cmd/displayswitch
    if ($LASTEXITCODE -ne 0) { throw "build failed" }
} finally {
    Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
    Pop-Location
}

$buildDir = Join-Path $root "build"
$stageDir = Join-Path $buildDir "display-switch-ss552-v$version"
$zipPath = "$stageDir.zip"
if (Test-Path $stageDir) { Remove-Item $stageDir -Recurse -Force }
if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
New-Item -ItemType Directory -Path $stageDir -Force | Out-Null

Copy-Item -Path $pluginDir -Destination $stageDir -Recurse
foreach ($file in @("install.ps1", "install.cmd", "uninstall.ps1", "uninstall.cmd", "README.md", "LICENSE")) {
    Copy-Item -Path (Join-Path $root $file) -Destination $stageDir
}
$hash = (Get-FileHash (Join-Path $pluginDir "displayswitch.exe") -Algorithm SHA256).Hash.ToLowerInvariant()
Set-Content -Path (Join-Path $stageDir "SHA256SUMS.txt") -Value "$hash  displayswitch.exe" -Encoding ASCII

Compress-Archive -Path $stageDir -DestinationPath $zipPath
Write-Host "Built $zipPath"
Write-Host "displayswitch.exe SHA-256: $hash"
