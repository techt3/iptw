# Windows Unblock Script for IPTW
# This script removes the "Zone.Identifier" that causes Windows Defender warnings
# Use this if you encounter SmartScreen warnings

param(
    [Parameter(Position=0)]
    [string]$BinaryPath = ""
)

# Function to show usage
function Show-Usage {
    Write-Host ""
    Write-Host "IPTW Windows Unblock Script" -ForegroundColor Cyan
    Write-Host "===========================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: .\windows-unblock.ps1 [path-to-iptw-binary]" -ForegroundColor White
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Yellow
    Write-Host "  .\windows-unblock.ps1                    # Auto-detect iptw.exe in current directory"
    Write-Host "  .\windows-unblock.ps1 iptw.exe           # Specify filename"
    Write-Host "  .\windows-unblock.ps1 C:\tools\iptw.exe  # Full path"
    Write-Host ""
    Write-Host "This script removes the Zone.Identifier that causes Windows Defender/SmartScreen warnings."
}

# Function to find binary
function Find-Binary {
    if ($BinaryPath -eq "") {
        # Try to find iptw.exe in common locations
        $candidates = @(".\iptw.exe", "..\iptw.exe", "iptw.exe")
        
        foreach ($candidate in $candidates) {
            if (Test-Path $candidate) {
                return Resolve-Path $candidate
            }
        }
        
        Write-Host "❌ Could not find iptw.exe automatically." -ForegroundColor Red
        Write-Host "   Please specify the path to the binary." -ForegroundColor Red
        Show-Usage
        exit 1
    } else {
        if (Test-Path $BinaryPath) {
            return Resolve-Path $BinaryPath
        } else {
            Write-Host "❌ Binary not found: $BinaryPath" -ForegroundColor Red
            exit 1
        }
    }
}

# Function to unblock file
function Unblock-Binary {
    param([string]$Path)
    
    Write-Host "🧹 Unblocking file: $Path" -ForegroundColor Yellow
    
    try {
        Unblock-File -Path $Path
        Write-Host "✅ File unblocked successfully!" -ForegroundColor Green
    }
    catch {
        Write-Host "ℹ️  No block found (file may already be trusted)" -ForegroundColor Blue
    }
}

# Function to verify binary
function Test-Binary {
    param([string]$Path)
    
    Write-Host "🔍 Verifying binary..." -ForegroundColor Yellow
    
    # Check if it's an executable
    $fileInfo = Get-Item $Path
    if ($fileInfo.Extension -eq ".exe") {
        Write-Host "✅ File appears to be a Windows executable" -ForegroundColor Green
    } else {
        Write-Host "⚠️  Warning: File doesn't have .exe extension" -ForegroundColor Yellow
    }
    
    # Try to run version command
    try {
        $version = & $Path --version 2>$null
        if ($version) {
            Write-Host "✅ Binary responds to version command: $($version.Split([Environment]::NewLine)[0])" -ForegroundColor Green
        }
    }
    catch {
        Write-Host "⚠️  Warning: Binary doesn't respond to --version command" -ForegroundColor Yellow
    }
}

# Main execution
function Main {
    Write-Host ""
    Write-Host "🪟 IPTW Windows Unblock Script" -ForegroundColor Cyan
    Write-Host "==============================" -ForegroundColor Cyan
    Write-Host ""
    
    # Handle help
    if ($args -contains "-h" -or $args -contains "--help" -or $args -contains "/?" -or $args -contains "/h") {
        Show-Usage
        exit 0
    }
    
    # Find and check binary
    $fullPath = Find-Binary
    Write-Host "Found binary: $fullPath" -ForegroundColor White
    Write-Host ""
    
    # Unblock file
    Unblock-Binary -Path $fullPath
    Write-Host ""
    
    # Verify binary
    Test-Binary -Path $fullPath
    Write-Host ""
    
    Write-Host "✅ Setup complete! You should now be able to run:" -ForegroundColor Green
    Write-Host "   $fullPath" -ForegroundColor White
    Write-Host ""
    Write-Host "If you still see SmartScreen warnings:" -ForegroundColor Yellow
    Write-Host "   1. Click 'More info' → 'Run anyway'" -ForegroundColor Yellow
    Write-Host "   2. Or add an exclusion in Windows Defender" -ForegroundColor Yellow
}

# Check execution policy
$policy = Get-ExecutionPolicy
if ($policy -eq "Restricted") {
    Write-Host "⚠️  PowerShell execution policy is Restricted" -ForegroundColor Yellow
    Write-Host "   You may need to run: Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser" -ForegroundColor Yellow
    Write-Host ""
}

# Run main function
Main
