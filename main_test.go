package main

import (
	"math"
	"testing"
	"time"
)

func TestProcessDataReport(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		temp        float64
		humidity    float64
		hasHumidity bool
		ok          bool
	}{
		{
			name: "temperature only",
			data: []byte{0xC4, 0x09},
			temp: 25.00,
			ok:   true,
		},
		{
			name:        "temperature and humidity",
			data:        []byte{0xC4, 0x09, 0x88, 0x13},
			temp:        25.00,
			humidity:    50.00,
			hasHumidity: true,
			ok:          true,
		},
		{
			name: "negative temperature",
			data: []byte{0x38, 0xFF},
			temp: -2.00,
			ok:   true,
		},
		{
			name: "short report",
			data: []byte{0x01},
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			temp, humidity, hasHumidity, ok := processDataReport(tt.data)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if temp != tt.temp {
				t.Fatalf("temp = %v, want %v", temp, tt.temp)
			}
			if humidity != tt.humidity {
				t.Fatalf("humidity = %v, want %v", humidity, tt.humidity)
			}
			if hasHumidity != tt.hasHumidity {
				t.Fatalf("hasHumidity = %v, want %v", hasHumidity, tt.hasHumidity)
			}
		})
	}
}

func TestValidatePeriod(t *testing.T) {
	tests := []struct {
		name    string
		period  float64
		want    time.Duration
		wantErr bool
	}{
		{name: "normal", period: 60, want: time.Minute},
		{name: "subsecond", period: 0.2, want: 200 * time.Millisecond},
		{name: "zero", period: 0, wantErr: true},
		{name: "negative", period: -1, wantErr: true},
		{name: "nan", period: math.NaN(), wantErr: true},
		{name: "infinity", period: math.Inf(1), wantErr: true},
		{name: "below timer precision", period: 1e-12, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validatePeriod(tt.period)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("duration = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeviceIntervalMilliseconds(t *testing.T) {
	tests := []struct {
		period float64
		want   uint32
	}{
		{period: 0.2, want: 200},
		{period: 0.0001, want: 1},
		{period: 1.999, want: 1999},
	}

	for _, tt := range tests {
		if got := deviceIntervalMilliseconds(tt.period); got != tt.want {
			t.Fatalf("deviceIntervalMilliseconds(%v) = %d, want %d", tt.period, got, tt.want)
		}
	}
}
