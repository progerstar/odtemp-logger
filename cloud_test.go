package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func validCloudToken() string {
	return "utx1_" + strings.Repeat("A", 86)
}

func TestEnvFloat64(t *testing.T) {
	const name = "ODTEMP_TEST_FLOAT"

	t.Setenv(name, "12.5")
	got, err := envFloat64(name, 60)
	if err != nil || got != 12.5 {
		t.Fatalf("envFloat64 = %v, %v; want 12.5, nil", got, err)
	}

	t.Setenv(name, "")
	got, err = envFloat64(name, 60)
	if err != nil || got != 60 {
		t.Fatalf("envFloat64 (empty) = %v, %v; want fallback 60, nil", got, err)
	}

	t.Setenv(name, "abc")
	if _, err = envFloat64(name, 60); err == nil {
		t.Fatal("expected error for non-numeric value")
	}
}

func TestValidateCloudDeviceID(t *testing.T) {
	tests := []struct {
		name    string
		device  string
		wantErr bool
	}{
		{name: "valid", device: "odtemp-1"},
		{name: "valid underscore", device: "room_01"},
		{name: "empty", device: "", wantErr: true},
		{name: "slash", device: "room/01", wantErr: true},
		{name: "too long", device: strings.Repeat("a", 65), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCloudDeviceID(tt.device)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateCloudWriteToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{name: "valid", token: validCloudToken()},
		{name: "empty", token: "", wantErr: true},
		{name: "wrong prefix", token: "read_" + strings.Repeat("A", 86), wantErr: true},
		{name: "short", token: "utx1_" + strings.Repeat("A", 85), wantErr: true},
		{name: "invalid char", token: "utx1_" + strings.Repeat("A", 85) + "+", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCloudWriteToken(tt.token)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCloudStateEndpoint(t *testing.T) {
	endpoint, err := cloudStateEndpoint("https://example.com/", "sensor_01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if endpoint != "https://example.com/state/sensor_01" {
		t.Fatalf("endpoint = %q", endpoint)
	}
}

func TestCloudPayload(t *testing.T) {
	payload := cloudPayload(SensorSample{Temperature: 23.5, Humidity: 65, HasHumidity: true})
	if payload.Type != 1 {
		t.Fatalf("type = %d, want 1", payload.Type)
	}
	if payload.Temperature != 23.5 {
		t.Fatalf("temperature = %v, want 23.5", payload.Temperature)
	}
	if payload.Humidity == nil || *payload.Humidity != 65 {
		t.Fatalf("humidity = %v, want 65", payload.Humidity)
	}

	payload = cloudPayload(SensorSample{Temperature: 23.5})
	if payload.Humidity != nil {
		t.Fatalf("humidity = %v, want nil", *payload.Humidity)
	}
}

func TestCloudClientPostSample(t *testing.T) {
	token := validCloudToken()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
			http.Error(w, "unexpected method", http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/state/sensor-01" {
			t.Errorf("path = %s, want /state/sensor-01", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusBadRequest)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			t.Errorf("Authorization = %q", got)
			http.Error(w, "unexpected authorization", http.StatusBadRequest)
			return
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
			http.Error(w, "unexpected content type", http.StatusBadRequest)
			return
		}

		var payload map[string]float64
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if payload["ty"] != 1 || payload["v1"] != 21.25 || payload["v2"] != 44.5 {
			t.Errorf("payload = %#v", payload)
			http.Error(w, "unexpected payload", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewCloudClient(CloudConfig{
		BaseURL:    server.URL,
		DeviceID:   "sensor-01",
		WriteToken: token,
	})
	if err != nil {
		t.Fatalf("NewCloudClient: %v", err)
	}

	err = client.PostSampleWithTimeout(SensorSample{Temperature: 21.25, Humidity: 44.5, HasHumidity: true})
	if err != nil {
		t.Fatalf("PostSampleWithTimeout: %v", err)
	}
}
