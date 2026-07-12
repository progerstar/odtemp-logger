//go:build nogui

package main

import "errors"

const guiSupported = false

type UI struct{}

func NewUI() (*UI, error) {
	return nil, errors.New("приложение собрано без GUI (build tag nogui)")
}

func (u *UI) SetOnClosed(callback func())                                 {}
func (u *UI) UpdateMeasurements(temp float64, humidity float64, has bool) {}
func (u *UI) ShowWaiting()                                                {}
func (u *UI) ShowDisconnected()                                           {}
func (u *UI) ShowConnectionLost()                                         {}
func (u *UI) Run()                                                        {}
