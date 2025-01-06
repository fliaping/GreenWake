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
APP_NAME="greenwake-guard"
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
echo "创建目录结构..."
mkdir -p "$DIST_DIR/usr/bin" || { echo "创建 bin 目录失败"; exit 1; }
mkdir -p "$DIST_DIR/usr/share/$APP_NAME/assets" || { echo "创建资源目录失败"; exit 1; }
mkdir -p "$DIST_DIR/usr/share/applications" || { echo "创建应用程序目录失败"; exit 1; }
mkdir -p "$DIST_DIR/usr/share/icons/hicolor/256x256/apps" || { echo "创建图标目录失败"; exit 1; }
mkdir -p "$INSTALLER_DIR" || { echo "创建安装程序目录失败"; exit 1; }

# 编译应用
echo "编译应用..."
cd "$PROJECT_ROOT" || { echo "切换到项目根目录失败"; exit 1; }

# 设置编译环境变量
export CGO_ENABLED=1
export GOARCH=$ARCH
if [ "$ARCH" = "arm64" ]; then
    export CC=aarch64-linux-gnu-gcc
    export CXX=aarch64-linux-gnu-g++
    export PKG_CONFIG_PATH=/usr/lib/aarch64-linux-gnu/pkgconfig
    export CGO_CFLAGS="-I/usr/aarch64-linux-gnu/include"
    export CGO_LDFLAGS="-L/usr/aarch64-linux-gnu/lib"
fi

# 添加编译选项
BUILD_FLAGS="-ldflags=-s -w -X main.Version=$VERSION"
if [ "$ARCH" = "arm64" ]; then
    BUILD_FLAGS="$BUILD_FLAGS -tags netgo"
fi

go build $BUILD_FLAGS -o "$DIST_DIR/usr/bin/$APP_NAME" ./cmd/guard || { echo "编译应用失败"; exit 1; }

# 检查编译结果
if [ ! -f "$DIST_DIR/usr/bin/$APP_NAME" ]; then
    echo "编译后的二进制文件不存在"
    exit 1
fi

# 复制资源文件
echo "复制资源文件..."
if [ -d "$PROJECT_ROOT/assets" ]; then
    cp -r "$PROJECT_ROOT/assets/lang" "$DIST_DIR/usr/share/$APP_NAME/assets/" || { echo "复制语言文件失败"; exit 1; }
    cp -r "$PROJECT_ROOT/assets/icons" "$DIST_DIR/usr/share/$APP_NAME/assets/" || { echo "复制图标文件失败"; exit 1; }
    # 复制应用图标
    cp "$PROJECT_ROOT/assets/icons/icon.png" "$DIST_DIR/usr/share/icons/hicolor/256x256/apps/$APP_NAME.png" || { echo "复制应用图标失败"; exit 1; }
else
    echo "错误: 没有找到assets目录: $PROJECT_ROOT/assets"
    exit 1
fi

# 创建 .desktop 文件
echo "创建 .desktop 文件..."
cat > "$DIST_DIR/usr/share/applications/$APP_NAME.desktop" << EOF || { echo "创建 .desktop 文件失败"; exit 1; }
[Desktop Entry]
Name=Greenwake Guard
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
mkdir -p "$DEB_DIR/DEBIAN" || { echo "创建 DEB 目录失败"; exit 1; }
cat > "$DEB_DIR/DEBIAN/control" << EOF || { echo "创建 DEB 控制文件失败"; exit 1; }
Package: greenwake-guard
Version: $VERSION
Section: utils
Priority: optional
Architecture: $ARCH
Depends: libc6
Maintainer: Fliaping <fliaping@gmail.com>
Description: Greenwake Guard - Prevent system from sleeping
 A utility to prevent system from sleeping with various wake strategies.
EOF

# 复制文件到 DEB 目录
cp -r "$DIST_DIR"/* "$DEB_DIR" || { echo "复制文件到 DEB 目录失败"; exit 1; }
dpkg-deb --build "$DEB_DIR" "$BUILD_DIR/${APP_NAME}_${VERSION}_${ARCH}.deb" || { echo "构建 DEB 包失败"; exit 1; }

# 创建 RPM 包
echo "创建 RPM 包..."
mkdir -p "$RPM_DIR/SPECS" || { echo "创建 RPM 目录失败"; exit 1; }

# 将 arm64 转换为 aarch64（RPM 使用的架构名称）
RPM_ARCH=$ARCH
if [ "$ARCH" = "arm64" ]; then
    RPM_ARCH="aarch64"
fi

cat > "$RPM_DIR/SPECS/$APP_NAME.spec" << EOF || { echo "创建 RPM spec 文件失败"; exit 1; }
Name:           $APP_NAME
Version:        $VERSION
Release:        1%{?dist}
Summary:        Prevent system from sleeping
License:        MIT
URL:            https://github.com/fliaping/GreenWake
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
         "$RPM_DIR/SPECS/$APP_NAME.spec" || { echo "构建 RPM 包失败"; exit 1; }

# 创建自安装脚本
echo "创建自安装脚本..."
cat > "$INSTALLER_DIR/header.sh" << 'EOF' || { echo "创建安装脚本头部失败"; exit 1; }
#!/bin/bash

# 提取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="$HOME/.local"
BIN_DIR="$INSTALL_DIR/bin"
SHARE_DIR="$INSTALL_DIR/share/greenwake-guard"
DESKTOP_DIR="$HOME/.local/share/applications"
ICONS_DIR="$HOME/.local/share/icons/hicolor/256x256/apps"

# 创建目录
mkdir -p "$BIN_DIR" || { echo "创建 bin 目录失败"; exit 1; }
mkdir -p "$SHARE_DIR" || { echo "创建共享目录失败"; exit 1; }
mkdir -p "$DESKTOP_DIR" || { echo "创建桌面文件目录失败"; exit 1; }
mkdir -p "$ICONS_DIR" || { echo "创建图标目录失败"; exit 1; }

# 解压二进制文件和资源
ARCHIVE=`awk '/^__ARCHIVE_BELOW__/ {print NR + 1; exit 0; }' "$0"`
tail -n+$ARCHIVE "$0" | tar xz -C "$INSTALL_DIR" || { echo "解压文件失败"; exit 1; }

# 创建桌面文件
cat > "$DESKTOP_DIR/greenwake-guard.desktop" << EOL || { echo "创建桌面文件失败"; exit 1; }
[Desktop Entry]
Name=Greenwake Guard
Comment=Prevent system from sleeping
Exec=$BIN_DIR/greenwake-guard
Icon=greenwake-guard
Terminal=false
Type=Application
Categories=Utility;
StartupNotify=false
EOL

echo "安装完成！"
echo "程序已安装到: $BIN_DIR/greenwake-guard"
echo "你可以运行 '$BIN_DIR/greenwake-guard' 来启动程序"
exit 0

__ARCHIVE_BELOW__
EOF

# 创建临时目录并准备文件
echo "准备安装文件..."
TEMP_DIR=$(mktemp -d) || { echo "创建临时目录失败"; exit 1; }
mkdir -p "$TEMP_DIR/bin" || { echo "创建临时 bin 目录失败"; exit 1; }
mkdir -p "$TEMP_DIR/share/greenwake-guard/assets" || { echo "创建临时 assets 目录失败"; exit 1; }
mkdir -p "$TEMP_DIR/share/icons/hicolor/256x256/apps" || { echo "创建临时图标目录失败"; exit 1; }

cp "$DIST_DIR/usr/bin/$APP_NAME" "$TEMP_DIR/bin/" || { echo "复制二进制文件失败"; exit 1; }
if [ -d "$DIST_DIR/usr/share/$APP_NAME/assets" ]; then
    cp -r "$DIST_DIR/usr/share/$APP_NAME/assets"/* "$TEMP_DIR/share/greenwake-guard/assets/" || { echo "复制资源文件失败"; exit 1; }
fi
if [ -f "$DIST_DIR/usr/share/icons/hicolor/256x256/apps/$APP_NAME.png" ]; then
    cp "$DIST_DIR/usr/share/icons/hicolor/256x256/apps/$APP_NAME.png" "$TEMP_DIR/share/icons/hicolor/256x256/apps/" || { echo "复制图标文件失败"; exit 1; }
fi

# 创建tar包并附加到脚本
cd "$TEMP_DIR" || { echo "切换到临时目录失败"; exit 1; }
tar czf - * >> "$INSTALLER_DIR/header.sh" || { echo "创建 tar 包失败"; exit 1; }

# 生成最终的安装脚本
mv "$INSTALLER_DIR/header.sh" "$BUILD_DIR/GreenwakeGuard_${VERSION}_${ARCH}.sh" || { echo "移动安装脚本失败"; exit 1; }
chmod +x "$BUILD_DIR/GreenwakeGuard_${VERSION}_${ARCH}.sh" || { echo "设置安装脚本权限失败"; exit 1; }

# 清理临时目录
rm -rf "$TEMP_DIR"

echo "打包完成！"
echo "DEB 包位置: $BUILD_DIR/${APP_NAME}_${VERSION}_${ARCH}.deb"
echo "RPM 包位置: $BUILD_DIR/${APP_NAME}-${VERSION}-1.${RPM_ARCH}.rpm"
echo "自安装脚本位置: $BUILD_DIR/GreenwakeGuard_${VERSION}_${ARCH}.sh" 