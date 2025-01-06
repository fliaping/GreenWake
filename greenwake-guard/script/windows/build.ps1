# Set error action to stop immediately
$ErrorActionPreference = "Stop"

# Get script directory absolute path
$SCRIPT_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path
# Get project root directory
$PROJECT_ROOT = (Get-Item $SCRIPT_DIR).Parent.Parent.FullName

# Set variables
$APP_NAME = "Greenwake Guard"
$VERSION = if ($env:VERSION) { $env:VERSION } else { "0.0.1" }  # Use default if VERSION not set
$ARCH = if ($env:GOARCH) { $env:GOARCH } else { "amd64" }  # Use default if GOARCH not set
$BUILD_DIR = Join-Path $PROJECT_ROOT "build\windows"
$DIST_DIR = Join-Path $BUILD_DIR "dist"
$RESOURCES_DIR = Join-Path $DIST_DIR "assets"

# Clean old build directory
if (Test-Path $BUILD_DIR) {
    Remove-Item -Recurse -Force $BUILD_DIR -ErrorAction Stop
}

# Create directory structure
Write-Host "Creating directory structure..."
try {
    New-Item -ItemType Directory -Force -Path $DIST_DIR -ErrorAction Stop | Out-Null
    New-Item -ItemType Directory -Force -Path $RESOURCES_DIR -ErrorAction Stop | Out-Null
} catch {
    Write-Host "Failed to create directories: $_"
    exit 1
}

# Build application
Write-Host "Building application..."
try {
    Push-Location $PROJECT_ROOT
    $env:CGO_ENABLED = "1"
    $env:GOOS = "windows"
    
    # Use Start-Process to capture go build errors
    $buildProcess = Start-Process -FilePath "go" -ArgumentList "build", "-ldflags", "`"-s -w -H=windowsgui -X main.Version=$VERSION`"", "-o", "$DIST_DIR\greenwake-guard.exe", ".\cmd\guard" -Wait -NoNewWindow -PassThru
    if ($buildProcess.ExitCode -ne 0) {
        throw "Failed to build application"
    }
} catch {
    Write-Host "Build failed: $_"
    exit 1
} finally {
    Pop-Location
}

# Copy resource files
Write-Host "Copying resource files..."
try {
    if (Test-Path (Join-Path $PROJECT_ROOT "assets")) {
        # Create resource directories
        New-Item -ItemType Directory -Force -Path (Join-Path $RESOURCES_DIR "lang") -ErrorAction Stop | Out-Null
        New-Item -ItemType Directory -Force -Path (Join-Path $RESOURCES_DIR "icons") -ErrorAction Stop | Out-Null
        
        # Copy language files
        Copy-Item (Join-Path $PROJECT_ROOT "assets\lang\*.json") (Join-Path $RESOURCES_DIR "lang") -ErrorAction Stop
        # Copy icon files
        Copy-Item (Join-Path $PROJECT_ROOT "assets\icons\*") (Join-Path $RESOURCES_DIR "icons") -ErrorAction Stop
    } else {
        throw "Assets directory not found: $PROJECT_ROOT\assets"
    }
} catch {
    Write-Host "Failed to copy resource files: $_"
    exit 1
}

# Create Inno Setup script
$SETUP_SCRIPT = @"
#define MyAppName "Greenwake Guard"
#define MyAppVersion "$VERSION"
#define MyAppPublisher "Fliaping"
#define MyAppURL "https://github.com/fliaping/GreenWake"
#define MyAppExeName "greenwake-guard.exe"

[Setup]
AppId={{新的GUID}}
AppName=Greenwake Guard
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
AllowNoIcons=yes
OutputDir=$BUILD_DIR
OutputBaseFilename=GreenwakeGuard_Setup_{#MyAppVersion}_$ARCH
Compression=lzma
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=$ARCH
ArchitecturesInstallIn64BitMode=$ARCH

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "chinesesimplified"; MessagesFile: "compiler:Languages\ChineseSimplified.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "autostart"; Description: "Auto start with Windows"; GroupDescription: "Additional options:"; Flags: unchecked

[Files]
Source: "$DIST_DIR\greenwake-guard.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "$DIST_DIR\assets\*"; DestDir: "{app}\assets"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Registry]
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "{#MyAppName}"; ValueData: """{app}\{#MyAppExeName}"""; Flags: uninsdeletevalue; Tasks: autostart

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent
"@

try {
    # Save Inno Setup script
    $SETUP_SCRIPT | Out-File -Encoding UTF8 (Join-Path $BUILD_DIR "setup.iss") -ErrorAction Stop

    # Check if Inno Setup is installed
    $INNO_SETUP = (Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Inno Setup 6_is1" -ErrorAction SilentlyContinue).InstallLocation
    if (-not $INNO_SETUP) {
        throw "Inno Setup not found. Please install Inno Setup 6: https://jrsoftware.org/isdl.php"
    }

    # Build installer
    Write-Host "Creating installer..."
    $ISCC = Join-Path $INNO_SETUP "ISCC.exe"
    $compileProcess = Start-Process -FilePath $ISCC -ArgumentList (Join-Path $BUILD_DIR "setup.iss") -Wait -NoNewWindow -PassThru
    if ($compileProcess.ExitCode -ne 0) {
        throw "Failed to create installer"
    }

    Write-Host "Build completed successfully!"
    Write-Host "Installer location: $BUILD_DIR\GreenwakeGuard_Setup_${VERSION}_${ARCH}.exe"
} catch {
    Write-Host "Failed to create installer: $_"
    exit 1
} 