$VERSION = "1.0.0"
$BINARY_NAME = "chaosrunner"
$REPO_URL = "https://github.com/dablon/chaosrunner"

Write-Host "Installing $BINARY_NAME v$VERSION..." -ForegroundColor Cyan

# Detect OS
$OS = $PSVersionTable.OS
if ($OS -match "Linux") { $OS_NAME = "linux" }
elseif ($OS -match "Darwin") { $OS_NAME = "darwin" }
else { $OS_NAME = "windows" }

# Detect Architecture
$ARCH = $env:PROCESSOR_ARCHITECTURE
if ($ARCH -eq "AMD64") { $ARCH_NAME = "amd64" }
elseif ($ARCH -eq "ARM64") { $ARCH_NAME = "arm64" }
else { $ARCH_NAME = $ARCH }

# Download URL
$DOWNLOAD_URL = "$REPO_URL/releases/latest/download/$BINARY_NAME-$OS_NAME-$ARCH_NAME"

# For Windows, add .exe
if ($OS_NAME -eq "windows") {
    $BINARY_NAME = "$BINARY_NAME.exe"
    $DOWNLOAD_URL = "$DOWNLOAD_URL.exe"
}

# Install directory
$INSTALL_DIR = "$env:LOCALAPPDATA\Programs\$BINARY_NAME"
if (-not (Test-Path $INSTALL_DIR)) {
    New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
}

# Download
Write-Host "Downloading from $DOWNLOAD_URL..." -ForegroundColor Yellow
Invoke-WebRequest -Uri $DOWNLOAD_URL -OutFile "$INSTALL_DIR\$BINARY_NAME" -UseBasicParsing

# Make executable (Linux/Mac)
if ($OS_NAME -ne "windows") {
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
}

# Add to PATH
$PATH_ENTRY = $INSTALL_DIR
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$PATH_ENTRY*") {
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$PATH_ENTRY", "User")
    Write-Host "Added to PATH: $PATH_ENTRY" -ForegroundColor Green
    Write-Host "Please restart your terminal or run: " -ForegroundColor Yellow
    Write-Host "  `$env:Path = [System.Environment]::GetEnvironmentVariable('Path','User')" -ForegroundColor Cyan
}

Write-Host "Installed to $INSTALL_DIR\$BINARY_NAME" -ForegroundColor Green
Write-Host "Done! Run: $BINARY_NAME --help" -ForegroundColor Cyan
