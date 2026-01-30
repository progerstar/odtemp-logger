package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

// UI содержит компоненты графического интерфейса
type UI struct {
	app      fyne.App
	window   fyne.Window
	tempText *canvas.Text
}

// NewUI создает новый графический интерфейс
// Возвращает nil и ошибку если GUI недоступен (нет драйверов/дисплея)
func NewUI() (ui *UI, err error) {
	// Перехватываем панику от fyne если нет драйверов/дисплея
	defer func() {
		if r := recover(); r != nil {
			ui = nil
			err = fmt.Errorf("не удалось инициализировать GUI: %v", r)
		}
	}()

	a := app.New()
	w := a.NewWindow("Монитор температуры")

	tempText := canvas.NewText("Поиск устройства...", theme.ForegroundColor())
	tempText.TextSize = 36
	tempText.TextStyle = fyne.TextStyle{Bold: true}
	tempText.Alignment = fyne.TextAlignCenter

	content := container.New(layout.NewCenterLayout(), tempText)
	w.SetContent(content)
	w.Resize(fyne.NewSize(400, 300))
	w.Show()

	return &UI{
		app:      a,
		window:   w,
		tempText: tempText,
	}, nil
}

// SetOnClosed устанавливает обработчик закрытия окна
func (u *UI) SetOnClosed(callback func()) {
	u.window.SetOnClosed(callback)
}

// UpdateTemperature обновляет отображение температуры
func (u *UI) UpdateTemperature(temp float64) {
	u.tempText.Text = fmt.Sprintf("%.1f°C", temp)
	u.tempText.TextSize = 72
	u.tempText.Color = theme.ForegroundColor()
	u.tempText.Refresh()
}

// ShowDisconnected показывает статус отключения
func (u *UI) ShowDisconnected() {
	u.tempText.Text = "Устройство отключено!"
	u.tempText.TextSize = 36
	u.tempText.Color = theme.ErrorColor()
	u.tempText.Refresh()
}

// ShowConnectionLost показывает статус потери связи
func (u *UI) ShowConnectionLost() {
	u.tempText.Text = "Потеряна связь!\nПоиск..."
	u.tempText.TextSize = 36
	u.tempText.Color = theme.ErrorColor()
	u.tempText.Refresh()
}

// Run запускает главный цикл приложения
func (u *UI) Run() {
	u.app.Run()
}
