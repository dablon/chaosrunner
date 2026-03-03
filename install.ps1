$VERSION = "1.0.0"
$BINARY_NAME = "chaosrunner"

$INSTALL_DIR = "$env:LOCALAPPDATA\Programs\$BINARY_NAME"
$BINARY_NAME_FINAL = "$BINARY_NAME.exe"

if (-not (Test-Path $INSTALL_DIR)) {
    New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
}

$SCRIPT_DIR = $PSScriptRoot
if (-not $SCRIPT_DIR) {
    $SCRIPT_DIR = Get-Location
}

$machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$env:Path = "$machinePath;$userPath"

Write-Host "Installing $BINARY_NAME v$VERSION from local source..." -ForegroundColor Cyan
Write-Host "Building with Go..." -ForegroundColor Yellow

Set-Location $SCRIPT_DIR
go build -buildvcs=false -o "$INSTALL_DIR\$BINARY_NAME_FINAL" ./cmd

if (-not (Test-Path "$INSTALL_DIR\$BINARY_NAME_FINAL")) {
    Write-Host "ERROR: Build failed" -ForegroundColor Red
    exit 1
}

Write-Host "Built successfully!" -ForegroundColor Green

$PATH_ENTRY = $INSTALL_DIR
if ($userPath -notlike "*$PATH_ENTRY*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$PATH_ENTRY", "User")
    $userPath = "$userPath;$PATH_ENTRY"
    Write-Host "Added to PATH: $PATH_ENTRY" -ForegroundColor Green
    
    if (-not ("Win32.NativeMethods" -as [type])) {
        Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition @"
            [DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
            public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
"@
        $HWND_BROADCAST = [IntPtr]0xffff
        $WM_SETTINGCHANGE = 0x1a
        $result = [UIntPtr]::Zero
        [Win32.NativeMethods]::SendMessageTimeout($HWND_BROADCAST, $WM_SETTINGCHANGE, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result) | Out-Null
    }
    
    $env:Path = "$machinePath;$userPath"
}

Write-Host "Installed to $INSTALL_DIR\$BINARY_NAME_FINAL" -ForegroundColor Green
Write-Host "Done! Run: $BINARY_NAME_FINAL --help" -ForegroundColor Cyan
