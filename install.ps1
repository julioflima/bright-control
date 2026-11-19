# BrightControl installer for Windows
#   irm https://raw.githubusercontent.com/julioflima/bright-control/main/install.ps1 | iex
$ErrorActionPreference = 'Stop'
$repo = 'julioflima/bright-control'
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
$dir  = Join-Path $env:LOCALAPPDATA 'BrightControl'
$exe  = Join-Path $dir 'BrightControl.exe'

Write-Host "Instalando BrightControl em $dir..."
New-Item -ItemType Directory -Force -Path $dir | Out-Null
Get-Process BrightControl -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep -Milliseconds 500
Invoke-WebRequest "https://github.com/$repo/releases/latest/download/brightcontrol-windows-$arch.exe" -OutFile $exe -UseBasicParsing
Unblock-File $exe
# The app registers itself to start with Windows (HKCU\...\Run) on first run.
Start-Process $exe
Write-Host "Pronto! Procure o icone da lua na bandeja do sistema (perto do relogio)."
