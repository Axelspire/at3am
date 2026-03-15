$ErrorActionPreference = "Stop"

$REPO = "Axelspire/at3am"

$ARCH = if ([Environment]::Is64BitProcess) { "amd64" } else { "arm64" }
$PLATFORM = "windows-$ARCH"

Write-Host "→ Downloading at3am (latest) for $PLATFORM..." -ForegroundColor Cyan

$URL = "https://github.com/$REPO/releases/latest/download/$PLATFORM.zip"

$ZIP = "$env:TEMP\at3am.zip"

Invoke-WebRequest -Uri $URL -OutFile $ZIP -UseBasicParsing

$INSTALL_DIR = "$env:USERPROFILE\.local\bin"
New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null

Expand-Archive -Path $ZIP -DestinationPath $INSTALL_DIR -Force

Get-ChildItem $INSTALL_DIR -Recurse -Filter "at3am*" | ForEach-Object {
    if (-not $_.PSIsContainer) { Move-Item $_.FullName $INSTALL_DIR -Force }
}

Write-Host "✅ at3am installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "   Run:        at3am --help"
Write-Host "   Add to PATH permanently:"
Write-Host "   [Environment]::SetEnvironmentVariable('Path', `"$INSTALL_DIR;`" + [Environment]::GetEnvironmentVariable('Path','User'),'User')"
