package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
	"time"
)

type fakeFeatureReportDevice struct {
	report       []byte
	reportLength int
	sent         []byte
}

func (d *fakeFeatureReportDevice) GetFeatureReport(buf []byte) (int, error) {
	copy(buf, d.report)
	return d.reportLength, nil
}

func (d *fakeFeatureReportDevice) SendFeatureReport(buf []byte) (int, error) {
	d.sent = append([]byte(nil), buf...)
	return len(buf), nil
}

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
		{name: "minimum", period: 0.001, want: time.Millisecond},
		{name: "zero", period: 0, wantErr: true},
		{name: "negative", period: -1, wantErr: true},
		{name: "nan", period: math.NaN(), wantErr: true},
		{name: "infinity", period: math.Inf(1), wantErr: true},
		{name: "below device resolution", period: 0.000999, wantErr: true},
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

func TestDeviceSamplePeriod(t *testing.T) {
	tests := []struct {
		name         string
		logPeriod    time.Duration
		cloudPeriod  time.Duration
		cloudEnabled bool
		want         time.Duration
	}{
		{
			name:      "local only",
			logPeriod: 1500 * time.Millisecond,
			want:      1500 * time.Millisecond,
		},
		{
			name:         "slower cloud",
			logPeriod:    1500 * time.Millisecond,
			cloudPeriod:  2 * time.Second,
			cloudEnabled: true,
			want:         1500 * time.Millisecond,
		},
		{
			name:         "faster cloud",
			logPeriod:    1500 * time.Millisecond,
			cloudPeriod:  200 * time.Millisecond,
			cloudEnabled: true,
			want:         200 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deviceSamplePeriod(tt.logPeriod, tt.cloudPeriod, tt.cloudEnabled)
			if got != tt.want {
				t.Fatalf("deviceSamplePeriod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSearchDeviceStopsWhenAlreadyCanceled(t *testing.T) {
	quit := make(chan struct{})
	close(quit)

	if searchDevice(&DeviceState{}, quit, true) {
		t.Fatal("searchDevice returned success after cancellation")
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

func TestSetDeviceIntervalUsesActualFeatureReportLength(t *testing.T) {
	report := make([]byte, 16)
	report[0] = HID_CMD_REPORT_ID
	binary.LittleEndian.PutUint32(report[1:5], 5000)
	for i := 5; i < len(report); i++ {
		report[i] = byte(i)
	}

	dev := &fakeFeatureReportDevice{
		report:       report,
		reportLength: len(report),
	}
	if err := setDeviceInterval(dev, 200); err != nil {
		t.Fatalf("setDeviceInterval() error = %v", err)
	}
	if len(dev.sent) != len(report) {
		t.Fatalf("sent report length = %d, want %d", len(dev.sent), len(report))
	}
	if got := binary.LittleEndian.Uint32(dev.sent[1:5]); got != 200 {
		t.Fatalf("sent interval = %d, want 200", got)
	}
	if !bytes.Equal(dev.sent[5:], report[5:]) {
		t.Fatalf("non-interval feature data changed: got %v, want %v", dev.sent[5:], report[5:])
	}
}

func TestSetDeviceIntervalRejectsInvalidFeatureReportLength(t *testing.T) {
	for _, reportLength := range []int{4, 65} {
		t.Run(fmt.Sprintf("length_%d", reportLength), func(t *testing.T) {
			dev := &fakeFeatureReportDevice{
				report:       make([]byte, 64),
				reportLength: reportLength,
			}
			if err := setDeviceInterval(dev, 200); err == nil {
				t.Fatal("setDeviceInterval() expected error")
			}
			if dev.sent != nil {
				t.Fatalf("unexpected feature report write: %v", dev.sent)
			}
		})
	}
}
