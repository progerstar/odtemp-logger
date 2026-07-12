package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultCloudBaseURL  = "https://cloud.unitx.pro"
	defaultCloudDeviceID = "odtemp-1"
	defaultCloudTimeout  = 10 * time.Second
)

type CloudConfig struct {
	BaseURL    string
	DeviceID   string
	WriteToken string
}

type CloudClient struct {
	endpoint string
	token    string
	client   *http.Client
}

type cloudStatePayload struct {
	Type        int      `json:"ty"`
	Temperature float64  `json:"v1"`
	Humidity    *float64 `json:"v2,omitempty"`
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envFloat64(name string, fallback float64) (float64, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %q не является числом", name, value)
	}
	return parsed, nil
}

func validateCloudDeviceID(deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("device_id не должен быть пустым")
	}
	if len(deviceID) > 64 {
		return fmt.Errorf("device_id должен быть не длиннее 64 байт")
	}
	for i := 0; i < len(deviceID); i++ {
		ch := deviceID[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			continue
		}
		return fmt.Errorf("device_id может содержать только A-Z, a-z, 0-9, '_' и '-'")
	}
	return nil
}

func validateCloudWriteToken(token string) error {
	if token == "" {
		return fmt.Errorf("write_token не должен быть пустым")
	}
	if !strings.HasPrefix(token, "utx1_") || len(token) != 91 {
		return fmt.Errorf("write_token должен иметь формат utx1_ + 86 символов base64url")
	}
	for i := len("utx1_"); i < len(token); i++ {
		ch := token[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			continue
		}
		return fmt.Errorf("write_token содержит недопустимый символ")
	}
	return nil
}

func cloudStateEndpoint(baseURL string, deviceID string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", fmt.Errorf("cloud URL не должен быть пустым")
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("неверный cloud URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("cloud URL должен начинаться с http:// или https://")
	}
	if u.Host == "" {
		return "", fmt.Errorf("cloud URL должен содержать host")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("cloud URL не должен содержать query string или fragment")
	}

	trimmed := strings.TrimRight(u.String(), "/")
	return trimmed + "/state/" + url.PathEscape(deviceID), nil
}

func NewCloudClient(cfg CloudConfig) (*CloudClient, error) {
	if err := validateCloudDeviceID(cfg.DeviceID); err != nil {
		return nil, err
	}
	if err := validateCloudWriteToken(cfg.WriteToken); err != nil {
		return nil, err
	}

	endpoint, err := cloudStateEndpoint(cfg.BaseURL, cfg.DeviceID)
	if err != nil {
		return nil, err
	}

	return &CloudClient{
		endpoint: endpoint,
		token:    cfg.WriteToken,
		client:   &http.Client{},
	}, nil
}

func cloudPayload(sample SensorSample) cloudStatePayload {
	payload := cloudStatePayload{
		Type:        1,
		Temperature: sample.Temperature,
	}
	if sample.HasHumidity {
		humidity := sample.Humidity
		payload.Humidity = &humidity
	}
	return payload
}

func (c *CloudClient) PostSample(ctx context.Context, sample SensorSample) error {
	body, err := json.Marshal(cloudPayload(sample))
	if err != nil {
		return fmt.Errorf("ошибка формирования cloud payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ошибка создания cloud request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка cloud request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	io.Copy(io.Discard, resp.Body)
	if len(responseBody) > 0 {
		return fmt.Errorf("cloud вернул %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}
	return fmt.Errorf("cloud вернул %s", resp.Status)
}

func (c *CloudClient) PostSampleWithTimeout(sample SensorSample) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultCloudTimeout)
	defer cancel()
	return c.PostSample(ctx, sample)
}
