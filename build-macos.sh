#!/bin/bash
# Сборка для macOS
# Запускать на macOS

set -e

APP_NAME="ODTemp Logger"
APP_ID="com.opendev.odtemp-logger"
VERSION=$(grep 'VERSION.*=' main.go | head -1 | sed 's/.*"\(.*\)".*/\1/')

echo "Сборка $APP_NAME v$VERSION для macOS..."

# Проверка fyne CLI
if ! command -v fyne &> /dev/null; then
    echo "Установка fyne CLI..."
    go install fyne.io/tools/cmd/fyne@latest
fi

# Сборка .app bundle
fyne package -os darwin -name "$APP_NAME" -appID "$APP_ID" -icon Icon.png

# Создание DMG (опционально, если есть create-dmg)
if command -v create-dmg &> /dev/null; then
    echo "Создание DMG..."
    create-dmg \
        --volname "$APP_NAME" \
        --window-size 400 300 \
        --icon-size 100 \
        --app-drop-link 200 150 \
        "ODTemp_Logger_${VERSION}_macos.dmg" \
        "$APP_NAME.app"
fi

echo "Готово: $APP_NAME.app"
