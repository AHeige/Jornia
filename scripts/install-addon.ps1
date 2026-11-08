param([Parameter(Mandatory=$true)][string]$ClientDirectory)
$ErrorActionPreference = 'Stop'
# Explicit user-selected game directory only. No process inspection, registry access or privilege elevation.
$root = (Resolve-Path -LiteralPath $ClientDirectory).Path
if (!(Test-Path -LiteralPath (Join-Path $root 'WTF')) -and !(Test-Path -LiteralPath (Join-Path $root 'Interface'))) {
    throw 'Choose the actual Forever client directory containing Interface or WTF, not the Battle.net parent folder.'
}
$source = Join-Path $PSScriptRoot 'AddOns\Wisp'
if (!(Test-Path -LiteralPath (Join-Path $source 'Wisp_Camelot.toc'))) { throw 'Addon source is missing from this bundle.' }
$addons = Join-Path $root 'Interface\AddOns'
$target = Join-Path $addons 'Jornia'
New-Item -ItemType Directory -Force -Path $addons | Out-Null
if (Test-Path -LiteralPath $target) {
    $existing = Join-Path $target 'Wisp_Camelot.toc'
    if (!(Test-Path -LiteralPath $existing) -or !(Select-String -LiteralPath $existing -SimpleMatch 'Jornia contributors' -Quiet)) { throw 'An unrelated Jornia addon may already exist. Nothing was overwritten; choose a separate test client or inspect it manually.' }

    $backup = Join-Path $env:LOCALAPPDATA ('Wisp\addon-backups\Wisp-' + (Get-Date -Format 'yyyyMMdd-HHmmss-ffff'))
    New-Item -ItemType Directory -Force -Path (Split-Path $backup) | Out-Null
    Copy-Item -LiteralPath $target -Destination $backup -Recurse
    Write-Host "Previous addon backed up to $backup"
}
Copy-Item -LiteralPath $source -Destination $addons -Recurse -Force
Write-Host "Installed $target. Restart the client, enable Jornia and type /wisp."
