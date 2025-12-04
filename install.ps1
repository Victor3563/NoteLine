# install.ps1
$ErrorActionPreference = 'Stop'

# --- Config
$repo = "Victor3563/NoteLine"
$base = "https://github.com/$repo/releases/latest/download"
$zipName = "noteline-windows.zip"

try {
    Write-Host "Install: preparing..."

    # install dir
    $installDir = Join-Path $env:LOCALAPPDATA "Noteline"
    if (-not (Test-Path -Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir | Out-Null
    }

    $zipPath = Join-Path $installDir $zipName

    Write-Host "Downloading $zipName ..."
    Invoke-WebRequest -Uri "$base/$zipName" -OutFile $zipPath -UseBasicParsing

    Write-Host "Extracting..."
    Add-Type -AssemblyName System.IO.Compression.FileSystem

    # check existing
    $exePathCheck = Join-Path $installDir "noteline-windows-amd64.exe"
    $i18nPathCheck = Join-Path $installDir "i18n"

    $exists = (Test-Path -Path "$exePathCheck") -or (Test-Path -Path "$i18nPathCheck")
    if ($exists) {
        Write-Host "Existing files detected - removing old files..."
        # remove everything except the downloaded zip (if present)
        Get-ChildItem -Path $installDir -Force | Where-Object { $_.Name -ne $zipName } | ForEach-Object {
            try { Remove-Item -LiteralPath $_.FullName -Recurse -Force -ErrorAction SilentlyContinue } catch {}
        }
    }

    # Extract (will throw if zip missing or corrupt)
    [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $installDir)

    # find exe
    $exe = Get-ChildItem -Path $installDir -Filter "noteline*.exe" -File -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $exe) {
        Write-Error "Binary not found in archive."
        exit 1
    }

    # i18n folder (optional)
    $i18nDir = Join-Path $installDir "i18n"
    if (-not (Test-Path -Path $i18nDir)) {
        # leave $i18nDir as the expected location (may be empty)
        Write-Host "Warning: i18n folder not found; locales are optional."
    }

    # create user bin
    $userBin = Join-Path $env:USERPROFILE "bin"
    if (-not (Test-Path -Path $userBin)) { New-Item -ItemType Directory -Path $userBin | Out-Null }

    # remove old shims
    $oldFiles = @("noteline.cmd","noteline.ps1","noteline.bat","noteline.exe")
    foreach ($f in $oldFiles) {
        $p = Join-Path $userBin $f
        if (Test-Path -Path $p) { Remove-Item -LiteralPath $p -Force -ErrorAction SilentlyContinue }
    }

    # prepare shim content (CMD)
    $shimPath = Join-Path $userBin "noteline.cmd"
    $exePath = $exe.FullName

    # Ensure there are no stray double-quotes in paths
    $exePathEsc = $exePath -replace '"',''
    $i18nDirEsc = $i18nDir -replace '"',''

    $shimContent = @"
@echo off
REM Noteline shim created by install.ps1
set "NOTELINE_I18N_DIR=$i18nDirEsc"
"$exePathEsc" %*
"@

    # write shim as ASCII (compatible with CMD)
    Set-Content -Path $shimPath -Value $shimContent -Encoding ASCII

    # add userBin to current session PATH if missing
    $currentPathEntries = $env:PATH -split ';'
    if ($currentPathEntries -notcontains $userBin) {
        $env:PATH = "$userBin;$env:PATH"
    }

    # add userBin to user PATH permanently if missing
    $existing = [Environment]::GetEnvironmentVariable("PATH","User")
    $existsInUser = $false
    if (-not [string]::IsNullOrEmpty($existing)) {
        $existsInUser = ($existing -split ';') -contains $userBin
    }
    if (-not $existsInUser) {
        if ([string]::IsNullOrEmpty($existing)) {
            [Environment]::SetEnvironmentVariable("PATH", $userBin, "User")
        } else {
            [Environment]::SetEnvironmentVariable("PATH", ($existing + ";" + $userBin), "User")
        }
        Write-Host "Added $userBin to user PATH (will apply in new console windows)."
    }

    # cleanup zip
    Remove-Item -LiteralPath $zipPath -Force -ErrorAction SilentlyContinue

    Write-Host "Installed!"
    Write-Host "Binary: $exePath"
    Write-Host "Locales (i18n): $i18nDir"
    Write-Host "Shim: $shimPath"
    Write-Host "Note: For the new PATH to be visible, open a new console. You may run the binary directly now using its full path."

} catch {
    Write-Error "Install failed: $($_.Exception.Message)"
    exit 1
}
