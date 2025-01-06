#!/bin/bash

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# 获取项目根目录
PROJECT_ROOT="$SCRIPT_DIR/../.."

# 设置变量
APP_NAME="awake"
VERSION="${VERSION:-0.0.1}"  # 如果环境变量未设置，使用默认值
ARCH="${GOARCH:-amd64}"  # 如果未设置GOARCH，默认为amd64
BUILD_DIR="$PROJECT_ROOT/build/linux"
DIST_DIR="$BUILD_DIR/dist"
DEB_DIR="$BUILD_DIR/deb"
RPM_DIR="$BUILD_DIR/rpm"
INSTALLER_DIR="$BUILD_DIR/installer"

# 清理旧的构建目录
rm -rf "$BUILD_DIR"

# 创建目录结构
mkdir -p "$DIST_DIR/usr/bin"
mkdir -p "$DIST_DIR/usr/share/$APP_NAME/assets"
mkdir -p "$DIST_DIR/usr/share/applications"
mkdir -p "$DIST_DIR/usr/share/icons/hicolor/256x256/apps"
mkdir -p "$INSTALLER_DIR"

# 编译应用
echo "编译应用..."
cd "$PROJECT_ROOT"
CGO_ENABLED=1 GOARCH=$ARCH go build -ldflags "-s -w -X main.Version=$VERSION" -o "$DIST_DIR/usr/bin/$APP_NAME" cmd/awake/main.go

# 复制资源文件
echo "复制资源文件..."
if [ -d "$PROJECT_ROOT/assets" ]; then
    cp -r "$PROJECT_ROOT/assets/lang" "$DIST_DIR/usr/share/$APP_NAME/assets/"
    cp -r "$PROJECT_ROOT/assets/icons" "$DIST_DIR/usr/share/$APP_NAME/assets/"
    # 复制应用图标
    cp "$PROJECT_ROOT/assets/icons/icon.png" "$DIST_DIR/usr/share/icons/hicolor/256x256/apps/$APP_NAME.png"
else
    echo "警告: 没有找到assets目录: $PROJECT_ROOT/assets"
fi

# 创建 .desktop 文件
cat > "$DIST_DIR/usr/share/applications/$APP_NAME.desktop" << EOF
[Desktop Entry]
Name=Awake
Comment=Prevent system from sleeping
Exec=$APP_NAME
Icon=$APP_NAME
Terminal=false
Type=Application
Categories=Utility;
StartupNotify=false
EOF

# 创建 DEB 包
echo "创建 DEB 包..."
mkdir -p "$DEB_DIR/DEBIAN"
cat > "$DEB_DIR/DEBIAN/control" << EOF
Package: $APP_NAME
Version: $VERSION
Section: utils
Priority: optional
Architecture: $ARCH
Depends: libc6
Maintainer: Fliaping <fliaping@gmail.com>
Description: Prevent system from sleeping
 A utility to prevent system from sleeping with various wake strategies.
EOF

# 复制文件到 DEB 目录
cp -r "$DIST_DIR"/* "$DEB_DIR"
dpkg-deb --build "$DEB_DIR" "$BUILD_DIR/${APP_NAME}_${VERSION}_${ARCH}.deb"

# 创建 RPM 包
echo "创建 RPM 包..."
mkdir -p "$RPM_DIR/SPECS"

# 将 arm64 转换为 aarch64（RPM 使用的架构名称）
RPM_ARCH=$ARCH
if [ "$ARCH" = "arm64" ]; then
    RPM_ARCH="aarch64"
fi

cat > "$RPM_DIR/SPECS/$APP_NAME.spec" << EOF
Name:           $APP_NAME
Version:        $VERSION
Release:        1%{?dist}
Summary:        Prevent system from sleeping
License:        MIT
URL:            https://github.com/fliaping/awake
BuildArch:      $RPM_ARCH

%description
A utility to prevent system from sleeping with various wake strategies.

%install
cp -r $DIST_DIR/* %{buildroot}

%files
/usr/bin/$APP_NAME
/usr/share/$APP_NAME/assets/*
/usr/share/applications/$APP_NAME.desktop
/usr/share/icons/hicolor/256x256/apps/$APP_NAME.png
EOF

# 构建 RPM 包
rpmbuild -bb --define "_topdir $RPM_DIR" \
         --define "_rpmdir $BUILD_DIR" \
         --define "_builddir $BUILD_DIR" \
         --define "_buildrootdir $BUILD_DIR/.build" \
         --define "_sourcedir $BUILD_DIR" \
         --buildroot="$DIST_DIR" \
         "$RPM_DIR/SPECS/$APP_NAME.spec"

# 创建自安装脚本
echo "创建自安装脚本..."
cat > "$INSTALLER_DIR/header.sh" << 'EOF'
#!/bin/bash

# 提取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="$HOME/.local"
BIN_DIR="$INSTALL_DIR/bin"
SHARE_DIR="$INSTALL_DIR/share/awake"
DESKTOP_DIR="$HOME/.local/share/applications"
ICONS_DIR="$HOME/.local/share/icons/hicolor/256x256/apps"

# 创建目录
mkdir -p "$BIN_DIR"
mkdir -p "$SHARE_DIR"
mkdir -p "$DESKTOP_DIR"
mkdir -p "$ICONS_DIR"

# 解压二进制文件和资源
ARCHIVE=`awk '/^__ARCHIVE_BELOW__/ {print NR + 1; exit 0; }' "$0"`
tail -n+$ARCHIVE "$0" | tar xz -C "$INSTALL_DIR"

# 创建桌面文件
cat > "$DESKTOP_DIR/awake.desktop" << EOL
[Desktop Entry]
Name=Awake
Comment=Prevent system from sleeping
Exec=$BIN_DIR/awake
Icon=awake
Terminal=false
Type=Application
Categories=Utility;
StartupNotify=false
EOL

echo "安装完成！"
echo "程序已安装到: $BIN_DIR/awake"
echo "你可以运行 '$BIN_DIR/awake' 来启动程序"
exit 0

__ARCHIVE_BELOW__
EOF

# 创建临时目录并准备文件
TEMP_DIR=$(mktemp -d)
mkdir -p "$TEMP_DIR/bin"
mkdir -p "$TEMP_DIR/share/awake/assets"
cp "$DIST_DIR/usr/bin/$APP_NAME" "$TEMP_DIR/bin/"
cp -r "$DIST_DIR/usr/share/$APP_NAME/assets"/* "$TEMP_DIR/share/awake/assets/"
cp "$DIST_DIR/usr/share/icons/hicolor/256x256/apps/$APP_NAME.png" "$TEMP_DIR/share/icons/hicolor/256x256/apps/"

# 创建tar包并附加到脚本
cd "$TEMP_DIR"
tar czf - * >> "$INSTALLER_DIR/header.sh"

# 生成最终的安装脚本
mv "$INSTALLER_DIR/header.sh" "$BUILD_DIR/Awake_${VERSION}_${ARCH}.sh"
chmod +x "$BUILD_DIR/Awake_${VERSION}_${ARCH}.sh"

# 清理临时目录
rm -rf "$TEMP_DIR"

echo "打包完成！"
echo "DEB 包位置: $BUILD_DIR/${APP_NAME}_${VERSION}_${ARCH}.deb"
echo "RPM 包位置: $BUILD_DIR/${APP_NAME}-${VERSION}-1.${RPM_ARCH}.rpm"
echo "自安装脚本位置: $BUILD_DIR/Awake_${VERSION}_${ARCH}.sh" 