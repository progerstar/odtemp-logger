@echo off
REM Сборка для Windows
REM Запускать на Windows

setlocal enabledelayedexpansion

set APP_NAME=ODTemp Logger
set APP_ID=com.opendev.odtemp-logger

echo Сборка %APP_NAME% для Windows...

REM Проверка Go
where go >nul 2>nul
if %errorlevel% neq 0 (
    echo Ошибка: Go не установлен
    exit /b 1
)

REM Проверка fyne CLI
where fyne >nul 2>nul
if %errorlevel% neq 0 (
    echo Установка fyne CLI...
    go install fyne.io/fyne/v2/cmd/fyne@latest
)

REM Сборка .exe
echo Сборка бинарника...
go build -buildvcs=false -ldflags="-H windowsgui" -o odtemp-logger.exe .

REM Сборка с иконкой через fyne package
fyne package -os windows -name "%APP_NAME%" --app-id "%APP_ID%" -icon Icon.png

echo.
echo Готово: odtemp-logger.exe
echo.
pause
