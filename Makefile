APP_NAME := odtemp-logger
APP_ID := com.opendev.odtemp-logger
VERSION := $(shell grep 'VERSION.*=' main.go | head -1 | sed 's/.*"\(.*\)".*/\1/')
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

.PHONY: all build package clean run help

all: build

# Сборка бинарника для текущей платформы
build:
	@echo "Сборка $(APP_NAME) v$(VERSION) для $(GOOS)/$(GOARCH)..."
	go build -o $(APP_NAME)_$(GOOS)_$(GOARCH) .

# Сборка пакета через fyne
package:
	@echo "Создание пакета для $(GOOS)..."
	fyne package -os $(GOOS) -name "$(APP_NAME)" -appID "$(APP_ID)" -icon Icon.png

# Быстрая сборка и запуск
run:
	go run .

# Сборка CLI версии (без GUI по умолчанию)
cli:
	go build -o $(APP_NAME)_cli .
	@echo "Запуск: ./$(APP_NAME)_cli -cli"

# Очистка
clean:
	rm -f $(APP_NAME) $(APP_NAME)_*
	rm -rf *.app *.exe *.tar.xz *.dmg

# Показать переменные
info:
	@echo "APP_NAME: $(APP_NAME)"
	@echo "VERSION:  $(VERSION)"
	@echo "GOOS:     $(GOOS)"
	@echo "GOARCH:   $(GOARCH)"

help:
	@echo "Доступные команды:"
	@echo "  make build   - сборка бинарника"
	@echo "  make package - создание пакета (требует fyne CLI)"
	@echo "  make run     - запуск без сборки"
	@echo "  make cli     - сборка CLI версии"
	@echo "  make clean   - очистка"
	@echo "  make info    - информация о сборке"
