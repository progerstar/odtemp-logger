#!/bin/bash
# Сборка для Linux (x64 и arm64)
# Запускать на Linux

set -e

APP_NAME="odtemp-logger"
APP_ID="com.opendev.odtemp-logger"
VERSION=$(grep 'VERSION.*=' main.go | head -1 | sed 's/.*"\(.*\)".*/\1/')
ARCH=$(uname -m)

echo "Сборка $APP_NAME v$VERSION для Linux ($ARCH)..."

# Установка зависимостей (Debian/Ubuntu)
install_deps_debian() {
    sudo apt-get update
    sudo apt-get install -y \
        libgl1-mesa-dev \
        xorg-dev \
        libhidapi-dev \
        libhidapi-hidraw0
}

# Установка зависимостей (Fedora/RHEL)
install_deps_fedora() {
    sudo dnf install -y \
        mesa-libGL-devel \
        libXcursor-devel \
        libXrandr-devel \
        libXinerama-devel \
        libXi-devel \
        libXxf86vm-devel \
        hidapi-devel
}

# Проверка и установка зависимостей
if [ -f /etc/debian_version ]; then
    echo "Обнаружен Debian/Ubuntu..."
    if ! dpkg -l | grep -q libhidapi-dev; then
        echo "Установка зависимостей..."
        install_deps_debian
    fi
elif [ -f /etc/fedora-release ] || [ -f /etc/redhat-release ]; then
    echo "Обнаружен Fedora/RHEL..."
    if ! rpm -q hidapi-devel &> /dev/null; then
        echo "Установка зависимостей..."
        install_deps_fedora
    fi
fi

# Добавляем путь к Go binaries в PATH
export PATH="$PATH:$(go env GOPATH)/bin"

# Проверка fyne CLI
if ! command -v fyne &> /dev/null; then
    echo "Установка fyne CLI..."
    go install fyne.io/tools/cmd/fyne@latest
fi

# Определение архитектуры для имени файла
case $ARCH in
    x86_64)  ARCH_NAME="amd64" ;;
    aarch64) ARCH_NAME="arm64" ;;
    armv7l)  ARCH_NAME="arm" ;;
    *)       ARCH_NAME="$ARCH" ;;
esac

# Сборка бинарника
echo "Сборка бинарника..."
go build -o "${APP_NAME}_linux_${ARCH_NAME}" .

# Сборка tar.gz архива
fyne package -os linux -name "$APP_NAME" -appID "$APP_ID" -icon Icon.png

# Переименование с архитектурой
if [ -f "${APP_NAME}.tar.xz" ]; then
    mv "${APP_NAME}.tar.xz" "${APP_NAME}_${VERSION}_linux_${ARCH_NAME}.tar.xz"
    echo "Готово: ${APP_NAME}_${VERSION}_linux_${ARCH_NAME}.tar.xz"
fi

echo "Готово: ${APP_NAME}_linux_${ARCH_NAME}"
