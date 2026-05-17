# Crux Control Windows Installer (PowerShell)
# Supports Windows 10/11, amd64 and arm64

$ErrorActionPreference = "Stop"

$Repo = "danycrafts/crux"
$ApiUrl = "https://api.github.com/repos/$Repo/releases/latest"

# Detect architecture
$Arch = if ([Environment]::Is64BitOperatingSystem) {
    switch ($env:PROCESSOR_ARCHITECTURE) {
        "ARM64" { "arm64" }
        default { "amd64" }
    }
} else {
    "arm"
}

$OS = "windows"
$InstallDir = "$env:LOCALAPPDATA\Crux\bin"

Write-Host "==> Installing Crux Control..."
Write-Host "    OS: $OS"
Write-Host "    Arch: $Arch"
Write-Host "    Install dir: $InstallDir"

# Fetch version
$release = Invoke-RestMethod -Uri $ApiUrl -Headers @{ "User-Agent" = "crux-installer" }
$Version = $release.tag_name
if (-not $Version) {
    Write-Warning "Could not fetch latest release. Using v0.1.0"
    $Version = "v0.1.0"
}
Write-Host "    Version: $Version"

$BaseUrl = "https://github.com/$Repo/releases/download/$Version"
$TmpDir = [System.IO.Path]::GetTempPath() + [System.Guid]::NewGuid().ToString()
New-Item -ItemType Directory -Path $TmpDir | Out-Null

try {
    @("crux", "cruxd", "crux-dashboard") | ForEach-Object {
        $Bin = $_
        $File = "${Bin}_${OS}_${Arch}.exe"
        $Url = "$BaseUrl/$File"
        $Dest = "$InstallDir\$Bin.exe"

        Write-Host "==> Downloading $Bin..."
        try {
            Invoke-WebRequest -Uri $Url -OutFile "$TmpDir\$File" -UseBasicParsing
        } catch {
            Write-Warning "Failed to download $Bin from $Url"
            return
        }

        if (-not (Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }
        Move-Item -Path "$TmpDir\$File" -Destination $Dest -Force
        Write-Host "    Installed $Dest"
    }
} finally {
    Remove-Item -Recurse -Force $TmpDir -ErrorAction SilentlyContinue
}

# Add to PATH if needed
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "==> Added $InstallDir to your User PATH. Restart your terminal to use crux commands."
}

Write-Host "==> Installation complete."
Write-Host "    Run 'crux version' to verify."
Write-Host "    Run 'crux daemon start' to start the daemon."
Write-Host "    Run 'crux-dashboard' to open the web dashboard."
