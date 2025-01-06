#!/bin/bash

# 设置错误时立即退出
set -e
# 管道命令中的错误也会导致脚本退出
set -o pipefail
# 使用未定义的变量会报错
set -u

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# 获取项目根目录
PROJECT_ROOT="$SCRIPT_DIR/../.."

# 设置变量
APP_NAME="GreenwakeGuard"
VERSION="${VERSION:-0.0.1}"  # 如果环境变量未设置，使用默认值
BUILD_DIR="$PROJECT_ROOT/build/macos"
MACOS_DIR="$BUILD_DIR/$APP_NAME.app"
CONTENTS_DIR="$MACOS_DIR/Contents"
RESOURCES_DIR="$CONTENTS_DIR/Resources"
MACOS_BIN_DIR="$CONTENTS_DIR/MacOS"
TEMP_DIR="$BUILD_DIR/temp"

# 清理旧的构建目录
rm -rf "$BUILD_DIR"

# 创建目录结构
mkdir -p "$MACOS_BIN_DIR" || { echo "创建 MacOS 目录失败"; exit 1; }
mkdir -p "$RESOURCES_DIR/assets/lang" || { echo "创建资源目录失败"; exit 1; }
mkdir -p "$RESOURCES_DIR/assets/icons" || { echo "创建图标目录失败"; exit 1; }
mkdir -p "$TEMP_DIR" || { echo "创建临时目录失败"; exit 1; }

# 编译 Intel 版本
echo "编译 Intel (amd64) 版本..."
cd "$PROJECT_ROOT" || { echo "切换到项目根目录失败"; exit 1; }
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 CC="clang -arch x86_64" \
    go build -ldflags "-s -w -X main.Version=$VERSION" \
    -o "$TEMP_DIR/$APP_NAME-amd64" ./cmd/guard || { echo "编译 Intel 版本失败"; exit 1; }

# 编译 Apple Silicon 版本
echo "编译 Apple Silicon (arm64) 版本..."
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 CC="clang -arch arm64" \
    go build -ldflags "-s -w -X main.Version=$VERSION" \
    -o "$TEMP_DIR/$APP_NAME-arm64" ./cmd/guard || { echo "编译 Apple Silicon 版本失败"; exit 1; }

# 合并为 Universal Binary
echo "创建 Universal Binary..."
lipo -create -output "$MACOS_BIN_DIR/$APP_NAME" \
    "$TEMP_DIR/$APP_NAME-amd64" \
    "$TEMP_DIR/$APP_NAME-arm64" || { echo "创建 Universal Binary 失败"; exit 1; }

# 验证 Universal Binary
echo "验证 Universal Binary..."
lipo -info "$MACOS_BIN_DIR/$APP_NAME" || { echo "验证 Universal Binary 失败"; exit 1; }

# 清理临时文件
rm -rf "$TEMP_DIR"

# 复制资源文件
echo "复制资源文件..."
if [ -d "$PROJECT_ROOT/assets" ]; then
    cp -r "$PROJECT_ROOT/assets/lang" "$RESOURCES_DIR/assets/" || { echo "复制语言文件失败"; exit 1; }
    cp -r "$PROJECT_ROOT/assets/icons" "$RESOURCES_DIR/assets/" || { echo "复制图标文件失败"; exit 1; }
else
    echo "错误: 没有找到assets目录: $PROJECT_ROOT/assets"
    exit 1
fi

# 创建图标
if [ -f "$PROJECT_ROOT/assets/icons/icon.png" ]; then
    echo "创建应用图标..."
    ICONSET="$BUILD_DIR/$APP_NAME.iconset"
    mkdir -p "$ICONSET" || { echo "创建图标集目录失败"; exit 1; }
    
    # 生成不同尺寸的图标
    for size in 16 32 128 256 512; do
        sips -z $size $size "$PROJECT_ROOT/assets/icons/icon.png" --out "$ICONSET/icon_${size}x${size}.png" || { echo "生成 ${size}x${size} 图标失败"; exit 1; }
        if [ $size -lt 512 ]; then
            sips -z $((size*2)) $((size*2)) "$PROJECT_ROOT/assets/icons/icon.png" --out "$ICONSET/icon_${size}x${size}@2x.png" || { echo "生成 ${size}x${size}@2x 图标失败"; exit 1; }
        fi
    done
    
    # 转换为 icns 格式
    iconutil -c icns "$ICONSET" -o "$RESOURCES_DIR/$APP_NAME.icns" || { echo "转换为 icns 格式失败"; exit 1; }
    rm -rf "$ICONSET"
else
    echo "错误: 没有找到图标文件: $PROJECT_ROOT/assets/icons/icon.png"
    exit 1
fi

# 创建 Info.plist
cat > "$CONTENTS_DIR/Info.plist" << EOF || { echo "创建 Info.plist 失败"; exit 1; }
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>$APP_NAME</string>
    <key>CFBundleExecutable</key>
    <string>$APP_NAME</string>
    <key>CFBundleIdentifier</key>
    <string>com.fliaping.greenwakeguard</string>
    <key>CFBundleVersion</key>
    <string>$VERSION</string>
    <key>CFBundleShortVersionString</key>
    <string>$VERSION</string>
    <key>CFBundleIconFile</key>
    <string>$APP_NAME</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF

# 创建 DMG
echo "创建 DMG..."
DMG_NAME="${APP_NAME}_${VERSION}"
create-dmg \
    --volname "$APP_NAME" \
    --volicon "$RESOURCES_DIR/$APP_NAME.icns" \
    --window-pos 200 120 \
    --window-size 600 400 \
    --icon-size 100 \
    --icon "$APP_NAME.app" 175 190 \
    --hide-extension "$APP_NAME.app" \
    --app-drop-link 425 190 \
    --no-internet-enable \
    "$BUILD_DIR/$DMG_NAME.dmg" \
    "$MACOS_DIR" || { echo "创建 DMG 失败"; exit 1; }

# 重命名最终的 DMG 文件
mv "$BUILD_DIR/$DMG_NAME.dmg" "$BUILD_DIR/GreenwakeGuard.dmg" || { echo "重命名 DMG 文件失败"; exit 1; }

echo "打包完成！"
echo "DMG 包位置: $BUILD_DIR/GreenwakeGuard.dmg" 