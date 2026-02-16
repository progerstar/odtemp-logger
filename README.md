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
| `-bootloader` | Switch device to bootloader mode and exit |

### Examples

```bash
# Log every 30 seconds to custom directory
./odtemp-logger -cli -path /var/log/temp -period 30

# Fast polling (200ms)
./odtemp-logger -cli -period 0.2

# Silent mode (no log file)
./odtemp-logger -silent
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
| `-bootloader` | Перевести устройство в загрузчик и выйти |

### Примеры

```bash
# Логирование каждые 30 секунд в указанную директорию
./odtemp-logger -cli -path /var/log/temp -period 30

# Частый опрос (200мс)
./odtemp-logger -cli -period 0.2

# Без логирования
./odtemp-logger -silent
```
