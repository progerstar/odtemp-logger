# ODTemp Logger

Temperature and humidity monitoring application for USB sensor ODTEMP-1.

[Русский](#русский)

## Features

- Real-time temperature and humidity display
- CLI mode for headless systems
- Automatic device reconnection
- Logging to file
- Cross-platform: Windows, macOS, Linux

## Requirements

- USB temperature/humidity sensor ODTEMP-1
- Go 1.23+ (for building)

## Building

See [Build.md](Build.md) for platform-specific instructions.

```bash
# Build binary
make build

# Build package
make package

# Run without building
make run

# CLI-only build (nogui tag): no Fyne/OpenGL/X11 dependencies,
# suitable for headless servers; always runs in console mode
make cli
```

## Usage

### GUI Mode (default)

```bash
./odtemp-logger
```

### CLI Mode

```bash
./odtemp-logger -cli
```

### Command Line Options

| Flag | Description |
|------|-------------|
| `-cli` | Run without GUI |
| `-path <dir>` | Log file directory |
| `-silent` | Disable logging |
| `-period <sec>` | Log interval in seconds (default: 60) |
| `-cloud-token <write_token>` | Enable cloud-lite upload with write token |
| `-cloud-period <sec>` | Cloud upload interval in seconds (default: 60) |
| `-cloud-url <url>` | cloud-lite base URL (default: `https://cloud.unitx.pro`) |
| `-cloud-device <id>` | cloud-lite device_id (default: `odtemp-1`) |
| `-bootloader` | Switch device to bootloader mode and exit |

Cloud options can also be provided through the environment variables `ODTEMP_CLOUD_TOKEN`, `ODTEMP_CLOUD_PERIOD`, `ODTEMP_CLOUD_URL`, and `ODTEMP_CLOUD_DEVICE` (flags take precedence). Prefer `ODTEMP_CLOUD_TOKEN` over `-cloud-token`: command line arguments are visible to other users via `ps`.

### Examples

```bash
# Log every 30 seconds to custom directory
./odtemp-logger -cli -path /var/log/temp -period 30

# Fast polling (200ms)
./odtemp-logger -cli -period 0.2

# Silent mode (no log file)
./odtemp-logger -silent

# Upload readings to cloud-lite every 60 seconds
ODTEMP_CLOUD_TOKEN=utx1_... ./odtemp-logger -cli -cloud-period 60 -cloud-device room_01
```

---

# Русский

Приложение для мониторинга температуры/температуры-влажности с USB датчиков ODTEMP-1.

## Возможности

- Отображение температуры или температуры-влажности в реальном времени
- CLI режим для серверов
- Автоматическое переподключение устройства
- Логирование в файл
- Кроссплатформенность: Windows, macOS, Linux

## Требования

- USB датчик ODTEMP-1
- Go 1.23+ (для сборки)


## Сборка

```bash
# Собрать бинарник
make build

# Собрать пакет
make package

# Запустить без сборки
make run

# CLI-сборка (тег nogui): без зависимостей Fyne/OpenGL/X11,
# подходит для headless-серверов; всегда работает в консольном режиме
make cli
```

## Использование

### GUI режим (по умолчанию)

```bash
./odtemp-logger
```

### CLI режим

```bash
./odtemp-logger -cli
```

### Параметры командной строки

| Флаг | Описание |
|------|----------|
| `-cli` | Запуск без GUI |
| `-path <dir>` | Директория для лог-файлов |
| `-silent` | Отключить логирование |
| `-period <сек>` | Интервал записи в секундах (по умолчанию: 60) |
| `-cloud-token <write_token>` | Включить отправку в cloud-lite с write-токеном |
| `-cloud-period <сек>` | Интервал отправки в cloud-lite в секундах (по умолчанию: 60) |
| `-cloud-url <url>` | Base URL cloud-lite (по умолчанию: `https://cloud.unitx.pro`) |
| `-cloud-device <id>` | device_id в cloud-lite (по умолчанию: `odtemp-1`) |
| `-bootloader` | Перевести устройство в загрузчик и выйти |

Cloud-параметры также можно задать через переменные окружения `ODTEMP_CLOUD_TOKEN`, `ODTEMP_CLOUD_PERIOD`, `ODTEMP_CLOUD_URL` и `ODTEMP_CLOUD_DEVICE` (флаги имеют приоритет). Токен лучше передавать через `ODTEMP_CLOUD_TOKEN`, а не `-cloud-token`: аргументы командной строки видны другим пользователям через `ps`.

### Примеры

```bash
# Логирование каждые 30 секунд в указанную директорию
./odtemp-logger -cli -path /var/log/temp -period 30

# Частый опрос (200мс)
./odtemp-logger -cli -period 0.2

# Без логирования
./odtemp-logger -silent

# Отправка показаний в cloud-lite каждые 60 секунд
ODTEMP_CLOUD_TOKEN=utx1_... ./odtemp-logger -cli -cloud-period 60 -cloud-device room_01
```
