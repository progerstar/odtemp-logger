APP_NAME := odtemp-logger
APP_ID := com.opendev.odtemp-logger
VERSION := $(shell grep 'VERSION.*=' main.go | head -1 | sed 's/.*"\(.*\)".*/\1/')
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

.PHONY: all build package clean run help

all: build

build:
	@echo "Сборка $(APP_NAME) v$(VERSION) для $(GOOS)/$(GOARCH)..."
	go build -o $(APP_NAME)_$(GOOS)_$(GOARCH) .

package:
	@echo "Создание пакета для $(GOOS)..."
	fyne package -os $(GOOS) -name "$(APP_NAME)" -appID "$(APP_ID)" -icon Icon.png

run:
	go run .

cli:
	go build -tags nogui -o $(APP_NAME)_cli .
	@echo "Запуск: ./$(APP_NAME)_cli"

clean:
	rm -f $(APP_NAME) $(APP_NAME)_*
	rm -rf *.app *.exe *.tar.xz *.dmg

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
