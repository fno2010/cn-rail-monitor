package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	content := `
app:
  host: "127.0.0.1"
  port: 9090

query:
  interval: 60
  days_ahead: 3
  train_types:
    - "G"
    - "D"
  routes:
    - name: "北京到上海"
      from_station: "BJP"
      to_station: "SHH"

prometheus:
  enabled: true
  path: "/metrics"

telegraf:
  enabled: false

log:
  level: "debug"
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.App.Host != "127.0.0.1" {
		t.Errorf("App.Host = %q, want %q", cfg.App.Host, "127.0.0.1")
	}
	if cfg.App.Port != 9090 {
		t.Errorf("App.Port = %d, want %d", cfg.App.Port, 9090)
	}
	if cfg.Query.Interval != 60 {
		t.Errorf("Query.Interval = %d, want %d", cfg.Query.Interval, 60)
	}
	if cfg.Query.DaysAhead != 3 {
		t.Errorf("Query.DaysAhead = %d, want %d", cfg.Query.DaysAhead, 3)
	}
	if len(cfg.Query.Routes) != 1 {
		t.Fatalf("Routes length = %d, want 1", len(cfg.Query.Routes))
	}
	if cfg.Query.Routes[0].FromStation != "BJP" {
		t.Errorf("Route.FromStation = %q, want %q", cfg.Query.Routes[0].FromStation, "BJP")
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	content := `
query:
  routes:
    - name: "test"
      from_station: "BJP"
      to_station: "SHH"
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.App.Host != "0.0.0.0" {
		t.Errorf("App.Host = %q, want %q (default)", cfg.App.Host, "0.0.0.0")
	}
	if cfg.App.Port != 8080 {
		t.Errorf("App.Port = %d, want %d (default)", cfg.App.Port, 8080)
	}
	if cfg.Query.Interval != 300 {
		t.Errorf("Query.Interval = %d, want %d (default)", cfg.Query.Interval, 300)
	}
	if cfg.Query.DaysAhead != 5 {
		t.Errorf("Query.DaysAhead = %d, want %d (default)", cfg.Query.DaysAhead, 5)
	}
	if cfg.Prometheus.Path != "/metrics" {
		t.Errorf("Prometheus.Path = %q, want %q (default)", cfg.Prometheus.Path, "/metrics")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q (default)", cfg.Log.Level, "info")
	}
}

func TestLoadConfigInvalidFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	content := `
app:
  port: "not-a-number"
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	_, err := Load(tmpFile)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestGetQueryInterval(t *testing.T) {
	q := &QueryConfig{
		Interval: 300,
	}

	d := q.GetQueryInterval()
	if d != 300*time.Second {
		t.Errorf("GetQueryInterval() = %v, want %v", d, 300*time.Second)
	}
}

func TestGetDatesToQueryWithDaysAhead(t *testing.T) {
	q := &QueryConfig{
		DaysAhead: 3,
	}

	dates := q.GetDatesToQuery()
	if len(dates) != 3 {
		t.Fatalf("GetDatesToQuery() returned %d dates, want 3", len(dates))
	}

	now := time.Now()
	for i, d := range dates {
		expected := now.AddDate(0, 0, i+1)
		if d.Year() != expected.Year() || d.Month() != expected.Month() || d.Day() != expected.Day() {
			t.Errorf("date[%d] = %v, want %v", i, d, expected)
		}
	}
}

func TestGetDatesToQueryWithDateRange(t *testing.T) {
	q := &QueryConfig{
		StartDate: "2026-02-20",
		EndDate:   "2026-02-22",
	}

	dates := q.GetDatesToQuery()
	if len(dates) != 3 {
		t.Fatalf("GetDatesToQuery() returned %d dates, want 3", len(dates))
	}

	expectedDates := []string{"2026-02-20", "2026-02-21", "2026-02-22"}
	for i, d := range dates {
		if d.Format("2006-01-02") != expectedDates[i] {
			t.Errorf("date[%d] = %v, want %v", i, d.Format("2006-01-02"), expectedDates[i])
		}
	}
}

func TestGetDatesToQueryInvalidDateFormat(t *testing.T) {
	q := &QueryConfig{
		StartDate: "invalid",
		EndDate:   "2026-02-22",
		DaysAhead: 3,
	}

	dates := q.GetDatesToQuery()
	if len(dates) != 3 {
		t.Errorf("expected fallback to days_ahead (3 dates), got %d", len(dates))
	}
}

func TestGetDatesToQueryEndBeforeStart(t *testing.T) {
	q := &QueryConfig{
		StartDate: "2026-02-22",
		EndDate:   "2026-02-20",
		DaysAhead: 3,
	}

	dates := q.GetDatesToQuery()
	if len(dates) != 3 {
		t.Errorf("expected fallback to days_ahead (3 dates), got %d", len(dates))
	}
}

func TestMultipleRoutes(t *testing.T) {
	content := `
query:
  routes:
    - name: "北京到上海"
      from_station: "BJP"
      to_station: "SHH"
    - name: "广州到深圳"
      from_station: "GZQ"
      to_station: "SZP"
    - name: "杭州到南京"
      from_station: "HZH"
      to_station: "NJH"
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Query.Routes) != 3 {
		t.Fatalf("Routes length = %d, want 3", len(cfg.Query.Routes))
	}

	routes := []struct {
		name string
		from string
		to   string
	}{
		{"北京到上海", "BJP", "SHH"},
		{"广州到深圳", "GZQ", "SZP"},
		{"杭州到南京", "HZH", "NJH"},
	}

	for i, r := range routes {
		if cfg.Query.Routes[i].Name != r.name {
			t.Errorf("Route[%d].Name = %q, want %q", i, cfg.Query.Routes[i].Name, r.name)
		}
		if cfg.Query.Routes[i].FromStation != r.from {
			t.Errorf("Route[%d].FromStation = %q, want %q", i, cfg.Query.Routes[i].FromStation, r.from)
		}
		if cfg.Query.Routes[i].ToStation != r.to {
			t.Errorf("Route[%d].ToStation = %q, want %q", i, cfg.Query.Routes[i].ToStation, r.to)
		}
	}
}

func TestTrainTypes(t *testing.T) {
	content := `
query:
  train_types:
    - "G"
    - "D"
    - "K"
    - "Z"
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Query.TrainTypes) != 4 {
		t.Fatalf("TrainTypes length = %d, want 4", len(cfg.Query.TrainTypes))
	}

	expected := []string{"G", "D", "K", "Z"}
	for i, tt := range expected {
		if cfg.Query.TrainTypes[i] != tt {
			t.Errorf("TrainTypes[%d] = %q, want %q", i, cfg.Query.TrainTypes[i], tt)
		}
	}
}

func TestEnablePrice(t *testing.T) {
	content := `
query:
  enable_price: true
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Query.EnablePrice {
		t.Error("Query.EnablePrice = false, want true")
	}
}

func TestTelegrafConfig(t *testing.T) {
	content := `
telegraf:
  enabled: true
  output_mode: "file"
  output_path: "/var/log/telegraf/train.log"
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Telegraf.Enabled {
		t.Error("Telegraf.Enabled = false, want true")
	}
	if cfg.Telegraf.OutputMode != "file" {
		t.Errorf("Telegraf.OutputMode = %q, want %q", cfg.Telegraf.OutputMode, "file")
	}
	if cfg.Telegraf.OutputPath != "/var/log/telegraf/train.log" {
		t.Errorf("Telegraf.OutputPath = %q, want %q", cfg.Telegraf.OutputPath, "/var/log/telegraf/train.log")
	}
}

func TestStationConfig(t *testing.T) {
	content := `
station:
  cache_path: "/custom/path/stations.json"
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Station.CachePath != "/custom/path/stations.json" {
		t.Errorf("Station.CachePath = %q, want %q", cfg.Station.CachePath, "/custom/path/stations.json")
	}
}
