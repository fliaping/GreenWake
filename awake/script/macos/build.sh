#!/bin/bash

# 设置变量
APP_NAME="Awake"
VERSION="1.0.0"
BUILD_DIR="build/macos"
CONTENTS_DIR="$BUILD_DIR/$APP_NAME.app/Contents"
RESOURCES_DIR="$CONTENTS_DIR/Resources"
MACOS_DIR="$CONTENTS_DIR/MacOS"

# 清理旧的构建目录
rm -rf "$BUILD_DIR/$APP_NAME.app" "$BUILD_DIR/$APP_NAME.dmg"

# 创建目录结构
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR"

# 编译应用
echo "编译应用..."
CGO_ENABLED=1 go build -o "$MACOS_DIR/awake" -ldflags "-s -w" cmd/awake/main.go

# 复制资源文件
echo "复制资源文件..."
cp -r assets "$RESOURCES_DIR/"
cp "$BUILD_DIR/Info.plist" "$CONTENTS_DIR/"

# 如果有图标文件，复制图标
if [ -f "$BUILD_DIR/icon.icns" ]; then
    cp "$BUILD_DIR/icon.icns" "$RESOURCES_DIR/"
fi

# 设置权限
chmod +x "$MACOS_DIR/awake"

# 如果设置了签名证书，则进行签名
if [ ! -z "$APPLE_DEVELOPER_IDENTITY" ]; then
    echo "签名应用..."
    codesign --force --options runtime --sign "$APPLE_DEVELOPER_IDENTITY" "$BUILD_DIR/$APP_NAME.app"
    
    # 验证签名
    codesign --verify --verbose "$BUILD_DIR/$APP_NAME.app"
fi

# 创建 DMG
echo "创建 DMG..."
create-dmg \
    --volname "$APP_NAME" \
    --volicon "$RESOURCES_DIR/icon.icns" \
    --window-pos 200 120 \
    --window-size 600 400 \
    --icon-size 100 \
    --icon "$APP_NAME.app" 175 120 \
    --hide-extension "$APP_NAME.app" \
    --app-drop-link 425 120 \
    "$BUILD_DIR/$APP_NAME.dmg" \
    "$BUILD_DIR/$APP_NAME.app"

# 如果设置了签名证书，则签名 DMG
if [ ! -z "$APPLE_DEVELOPER_IDENTITY" ]; then
    echo "签名 DMG..."
    codesign --force --sign "$APPLE_DEVELOPER_IDENTITY" "$BUILD_DIR/$APP_NAME.dmg"
    
    # 如果设置了公证相关变量，则进行公证
    if [ ! -z "$APPLE_ID" ] && [ ! -z "$APPLE_ID_PASSWORD" ]; then
        echo "正在公证 DMG..."
        xcrun altool --notarize-app \
            --primary-bundle-id "com.fliaping.awake" \
            --username "$APPLE_ID" \
            --password "$APPLE_ID_PASSWORD" \
            --file "$BUILD_DIR/$APP_NAME.dmg"
    fi
fi

echo "打包完成！"
echo "DMG 文件位置: $BUILD_DIR/$APP_NAME.dmg" 