#!/bin/bash

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
mkdir -p "$MACOS_BIN_DIR"
mkdir -p "$RESOURCES_DIR/assets/lang"
mkdir -p "$RESOURCES_DIR/assets/icons"
mkdir -p "$TEMP_DIR"

# 编译 Intel 版本
echo "编译 Intel (amd64) 版本..."
cd "$PROJECT_ROOT"
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 CC="clang -arch x86_64" \
    go build -ldflags "-s -w -X main.Version=$VERSION" \
    -o "$TEMP_DIR/$APP_NAME-amd64" ./cmd/guard

# 编译 Apple Silicon 版本
echo "编译 Apple Silicon (arm64) 版本..."
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 CC="clang -arch arm64" \
    go build -ldflags "-s -w -X main.Version=$VERSION" \
    -o "$TEMP_DIR/$APP_NAME-arm64" ./cmd/guard

# 合并为 Universal Binary
echo "创建 Universal Binary..."
lipo -create -output "$MACOS_BIN_DIR/$APP_NAME" \
    "$TEMP_DIR/$APP_NAME-amd64" \
    "$TEMP_DIR/$APP_NAME-arm64"

# 验证 Universal Binary
echo "验证 Universal Binary..."
lipo -info "$MACOS_BIN_DIR/$APP_NAME"

# 清理临时文件
rm -rf "$TEMP_DIR"

# 复制资源文件
echo "复制资源文件..."
if [ -d "$PROJECT_ROOT/assets" ]; then
    cp -r "$PROJECT_ROOT/assets/lang" "$RESOURCES_DIR/assets/"
    cp -r "$PROJECT_ROOT/assets/icons" "$RESOURCES_DIR/assets/"
else
    echo "警告: 没有找到assets目录: $PROJECT_ROOT/assets"
fi

# 创建图标
if [ -f "$PROJECT_ROOT/assets/icons/icon.png" ]; then
    echo "创建应用图标..."
    ICONSET="$BUILD_DIR/$APP_NAME.iconset"
    mkdir -p "$ICONSET"
    
    # 生成不同尺寸的图标
    sips -z 16 16 "$PROJECT_ROOT/assets/icons/icon.png" --out "$ICONSET/icon_16x16.png"
    sips -z 32 32 "$PROJECT_ROOT/assets/icons/icon.png" --out "$ICONSET/icon_16x16@2x.png"
    sips -z 32 32 "$PROJECT_ROOT/assets/icons/icon.png" --out "$ICONSET/icon_32x32.png"
    sips -z 64 64 "$PROJECT_ROOT/assets/icons/icon.png" --out "$ICONSET/icon_32x32@2x.png"
    sips -z 128 128 "$PROJECT_ROOT/assets/icons/icon.png" --out "$ICONSET/icon_128x128.png"
    sips -z 256 256 "$PROJECT_ROOT/assets/icons/icon.png" --out "$ICONSET/icon_128x128@2x.png"
    sips -z 256 256 "$PROJECT_ROOT/assets/icons/icon.png" --out "$ICONSET/icon_256x256.png"
    sips -z 512 512 "$PROJECT_ROOT/assets/icons/icon.png" --out "$ICONSET/icon_256x256@2x.png"
    sips -z 512 512 "$PROJECT_ROOT/assets/icons/icon.png" --out "$ICONSET/icon_512x512.png"
    sips -z 1024 1024 "$PROJECT_ROOT/assets/icons/icon.png" --out "$ICONSET/icon_512x512@2x.png"
    
    # 转换为 icns 格式
    iconutil -c icns "$ICONSET" -o "$RESOURCES_DIR/$APP_NAME.icns"
    rm -rf "$ICONSET"
else
    echo "警告: 没有找到图标文件: $PROJECT_ROOT/assets/icons/icon.png"
fi

# 创建 Info.plist
cat > "$CONTENTS_DIR/Info.plist" << EOF
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
    "$MACOS_DIR"

# 重命名最终的 DMG 文件
mv "$BUILD_DIR/$DMG_NAME.dmg" "$BUILD_DIR/GreenwakeGuard.dmg"

echo "打包完成！"
echo "DMG 包位置: $BUILD_DIR/GreenwakeGuard.dmg" 