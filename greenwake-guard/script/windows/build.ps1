# 设置错误时立即退出
$ErrorActionPreference = "Stop"

# 获取脚本所在目录的绝对路径
$SCRIPT_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path
# 获取项目根目录
$PROJECT_ROOT = (Get-Item $SCRIPT_DIR).Parent.Parent.FullName

# 设置变量
$APP_NAME = "Greenwake Guard"
$VERSION = if ($env:VERSION) { $env:VERSION } else { "0.0.1" }  # 如果环境变量未设置，使用默认值
$ARCH = if ($env:GOARCH) { $env:GOARCH } else { "amd64" }  # 如果未设置GOARCH，默认为amd64
$BUILD_DIR = Join-Path $PROJECT_ROOT "build\windows"
$DIST_DIR = Join-Path $BUILD_DIR "dist"
$RESOURCES_DIR = Join-Path $DIST_DIR "assets"

# 清理旧的构建目录
if (Test-Path $BUILD_DIR) {
    Remove-Item -Recurse -Force $BUILD_DIR -ErrorAction Stop
}

# 创建目录结构
Write-Host "创建目录结构..."
try {
    New-Item -ItemType Directory -Force -Path $DIST_DIR -ErrorAction Stop | Out-Null
    New-Item -ItemType Directory -Force -Path $RESOURCES_DIR -ErrorAction Stop | Out-Null
} catch {
    Write-Host "创建目录失败: $_"
    exit 1
}

# 编译应用
Write-Host "编译应用..."
try {
    Push-Location $PROJECT_ROOT
    $env:CGO_ENABLED = "1"
    $env:GOOS = "windows"
    
    # 使用 Start-Process 来捕获 go build 的错误
    $buildProcess = Start-Process -FilePath "go" -ArgumentList "build", "-ldflags", "`"-s -w -H=windowsgui -X main.Version=$VERSION`"", "-o", "$DIST_DIR\greenwake-guard.exe", ".\cmd\guard" -Wait -NoNewWindow -PassThru
    if ($buildProcess.ExitCode -ne 0) {
        throw "编译应用失败"
    }
} catch {
    Write-Host "编译失败: $_"
    exit 1
} finally {
    Pop-Location
}

# 复制资源文件
Write-Host "复制资源文件..."
try {
    if (Test-Path (Join-Path $PROJECT_ROOT "assets")) {
        # 创建资源目录
        New-Item -ItemType Directory -Force -Path (Join-Path $RESOURCES_DIR "lang") -ErrorAction Stop | Out-Null
        New-Item -ItemType Directory -Force -Path (Join-Path $RESOURCES_DIR "icons") -ErrorAction Stop | Out-Null
        
        # 复制语言文件
        Copy-Item (Join-Path $PROJECT_ROOT "assets\lang\*.json") (Join-Path $RESOURCES_DIR "lang") -ErrorAction Stop
        # 复制图标文件
        Copy-Item (Join-Path $PROJECT_ROOT "assets\icons\*") (Join-Path $RESOURCES_DIR "icons") -ErrorAction Stop
    } else {
        throw "没有找到assets目录: $PROJECT_ROOT\assets"
    }
} catch {
    Write-Host "复制资源文件失败: $_"
    exit 1
}

# 创建 Inno Setup 脚本
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
Name: "autostart"; Description: "开机自动启动"; GroupDescription: "其他选项:"; Flags: unchecked

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
    # 保存 Inno Setup 脚本
    $SETUP_SCRIPT | Out-File -Encoding UTF8 (Join-Path $BUILD_DIR "setup.iss") -ErrorAction Stop

    # 检查是否安装了 Inno Setup
    $INNO_SETUP = (Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Inno Setup 6_is1" -ErrorAction SilentlyContinue).InstallLocation
    if (-not $INNO_SETUP) {
        throw "未安装 Inno Setup，请先安装 Inno Setup 6: https://jrsoftware.org/isdl.php"
    }

    # 编译安装包
    Write-Host "创建安装包..."
    $ISCC = Join-Path $INNO_SETUP "ISCC.exe"
    $compileProcess = Start-Process -FilePath $ISCC -ArgumentList (Join-Path $BUILD_DIR "setup.iss") -Wait -NoNewWindow -PassThru
    if ($compileProcess.ExitCode -ne 0) {
        throw "创建安装包失败"
    }

    Write-Host "打包完成！"
    Write-Host "安装包位置: $BUILD_DIR\GreenwakeGuard_Setup_${VERSION}_${ARCH}.exe"
} catch {
    Write-Host "创建安装包失败: $_"
    exit 1
} 