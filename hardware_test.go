package main

import (
	"os"
	"testing"

	"github.com/sstallion/go-hid"
)

func TestHardwareFeatureReportRoundTrip(t *testing.T) {
	if os.Getenv("ODTEMP_HARDWARE_TEST") != "1" {
		t.Skip("set ODTEMP_HARDWARE_TEST=1 to test a connected sensor")
	}

	dev, err := findAndOpenDevice()
	if err != nil {
		t.Fatal(err)
	}
	defer hid.Exit()
	defer dev.Close()

	before, err := getDeviceInterval(dev)
	if err != nil {
		t.Fatalf("read interval before write: %v", err)
	}
	if err := setDeviceInterval(dev, before); err != nil {
		t.Fatalf("write unchanged interval: %v", err)
	}
	after, err := getDeviceInterval(dev)
	if err != nil {
		t.Fatalf("read interval after write: %v", err)
	}
	if after != before {
		t.Fatalf("interval changed during round trip: before=%d after=%d", before, after)
	}

	t.Logf("feature report round trip succeeded with interval %d ms", after)
}
