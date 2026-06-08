package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- System Info API (read-only) ---

// SystemInfo represents system information.
type SystemInfo struct {
	Version       string  `json:"version"`
	Hostname      string  `json:"hostname"`
	PhysicalMem   int64   `json:"physmem"`
	Model         string  `json:"model"`
	Cores         int     `json:"cores"`
	Uptime        string  `json:"uptime"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	DateTime      struct {
		Year   int    `json:"year"`
		Month  int    `json:"month"`
		Day    int    `json:"day"`
		Hour   int    `json:"hour"`
		Minute int    `json:"minute"`
		Second int    `json:"second"`
		TZ     string `json:"timezone"`
	} `json:"datetime"`
	SystemSerial  string    `json:"system_serial"`
	SystemProduct string    `json:"system_product"`
	Timezone      string    `json:"timezone"`
	Loadavg       []float64 `json:"loadavg"`
}

// GetSystemInfo retrieves system information.
func (c *Client) GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	tflog.Trace(ctx, "GetSystemInfo start")

	resp, err := c.Get(ctx, "/system/info")
	if err != nil {
		return nil, fmt.Errorf("getting system info: %w", err)
	}

	var info SystemInfo
	if err := json.Unmarshal(resp, &info); err != nil {
		return nil, fmt.Errorf("parsing system info response: %w", err)
	}

	tflog.Trace(ctx, "GetSystemInfo success")
	return &info, nil
}
