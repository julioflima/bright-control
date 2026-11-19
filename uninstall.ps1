# irm https://raw.githubusercontent.com/julioflima/bright-control/main/uninstall.ps1 | iex
Get-Process BrightControl -ErrorAction SilentlyContinue | Stop-Process -Force
Remove-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Name BrightControl -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force (Join-Path $env:LOCALAPPDATA 'BrightControl') -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force (Join-Path $env:APPDATA 'BrightControl') -ErrorAction SilentlyContinue
Write-Host "BrightControl removido."
