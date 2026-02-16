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
	app          fyne.App
	window       fyne.Window
	tempText     *canvas.Text
	humidityText *canvas.Text
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
	w := a.NewWindow("Монитор ODTEMP-1")

	tempText := canvas.NewText("Поиск устройства...", theme.ForegroundColor())
	tempText.TextSize = 36
	tempText.TextStyle = fyne.TextStyle{Bold: true}
	tempText.Alignment = fyne.TextAlignCenter

	humidityText := canvas.NewText("", theme.ForegroundColor())
	humidityText.TextSize = 44
	humidityText.TextStyle = fyne.TextStyle{Bold: true}
	humidityText.Alignment = fyne.TextAlignCenter
	humidityText.Hide()

	values := container.NewVBox(tempText, humidityText)
	content := container.New(layout.NewCenterLayout(), values)
	w.SetContent(content)
	w.Resize(fyne.NewSize(400, 300))
	w.Show()

	return &UI{
		app:          a,
		window:       w,
		tempText:     tempText,
		humidityText: humidityText,
	}, nil
}

// SetOnClosed устанавливает обработчик закрытия окна
func (u *UI) SetOnClosed(callback func()) {
	u.window.SetOnClosed(callback)
}

// UpdateMeasurements обновляет отображение температуры и (опционально) влажности
func (u *UI) UpdateMeasurements(temp float64, humidity float64, hasHumidity bool) {
	u.tempText.Text = fmt.Sprintf("%.1f°C", temp)
	u.tempText.TextSize = 62
	u.tempText.Color = theme.ForegroundColor()
	u.tempText.Refresh()

	if hasHumidity {
		u.humidityText.Text = fmt.Sprintf("%.1f%%", humidity)
		u.humidityText.TextSize = 50
		u.humidityText.Color = theme.ForegroundColor()
		u.humidityText.Show()
	} else {
		u.humidityText.Hide()
	}
	u.humidityText.Refresh()
}

// ShowDisconnected показывает статус отключения
func (u *UI) ShowDisconnected() {
	u.tempText.Text = "Устройство отключено!"
	u.tempText.TextSize = 36
	u.tempText.Color = theme.ErrorColor()
	u.tempText.Refresh()

	u.humidityText.Text = ""
	u.humidityText.Hide()
	u.humidityText.Refresh()
}

// ShowConnectionLost показывает статус потери связи
func (u *UI) ShowConnectionLost() {
	u.tempText.Text = "Потеряна связь!"
	u.tempText.TextSize = 36
	u.tempText.Color = theme.ErrorColor()
	u.tempText.Refresh()

	u.humidityText.Text = "Поиск..."
	u.humidityText.TextSize = 32
	u.humidityText.Color = theme.ErrorColor()
	u.humidityText.Show()
	u.humidityText.Refresh()
}

// Run запускает главный цикл приложения
func (u *UI) Run() {
	u.app.Run()
}
