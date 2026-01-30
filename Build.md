# Сборка

| Файл              | Платформа       | Описание                                   |
  |-------------------|-----------------|--------------------------------------------|
  | build-macos.sh    | macOS           | Создаёт .app bundle, опционально DMG       |
  | build-linux.sh    | Linux x64/arm64 | Устанавливает зависимости, собирает tar.xz |
  | build-windows.bat | Windows         | Простой batch скрипт                       |
  | build-windows.ps1 | Windows         | PowerShell с версионированием              |
  | Makefile          | Любая           | Универсальная сборка       |

  Использование:
  # На текущей платформе
  make build      # только бинарник
  make package    # пакет через fyne

  # На macOS
  ./build-macos.sh

  # На Linux (x64 или arm64)
  ./build-linux.sh

  # На Windows
  .\build-windows.ps1
  # или
  build-windows.bat


