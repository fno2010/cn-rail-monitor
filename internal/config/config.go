package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	App        AppConfig        `yaml:"app"`
	Query      QueryConfig      `yaml:"query"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
	Telegraf   TelegrafConfig   `yaml:"telegraf"`
	Log        LogConfig        `yaml:"log"`
}

// AppConfig holds server configuration
type AppConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// QueryConfig holds query settings
type QueryConfig struct {
	Interval    int           `yaml:"interval"`
	DaysAhead   int           `yaml:"days_ahead"`
	StartDate   string        `yaml:"start_date"` // Specific start date (YYYY-MM-DD), takes precedence over days_ahead
	EndDate     string        `yaml:"end_date"`   // Specific end date (YYYY-MM-DD)
	TrainTypes  []string      `yaml:"train_types"`
	Routes      []RouteConfig `yaml:"routes"`
	EnablePrice bool          `yaml:"enable_price"` // Enable price monitoring (default: false)
}

// RouteConfig holds a single route configuration
type RouteConfig struct {
	Name         string `yaml:"name"`
	FromStation  string `yaml:"from_station"`
	ToStation    string `yaml:"to_station"`
	DateTemplate string `yaml:"date_template"`
}

// PrometheusConfig holds Prometheus settings
type PrometheusConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

// TelegrafConfig holds Telegraf output settings
type TelegrafConfig struct {
	Enabled    bool   `yaml:"enabled"`
	OutputMode string `yaml:"output_mode"` // "file" or "stdout"
	OutputPath string `yaml:"output_path"`
}

// LogConfig holds logging settings
type LogConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	cfg.applyDefaults()

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.App.Host == "" {
		c.App.Host = "0.0.0.0"
	}
	if c.App.Port == 0 {
		c.App.Port = 8080
	}
	if c.Query.Interval == 0 {
		c.Query.Interval = 300 // 5 minutes
	}
	if c.Query.DaysAhead == 0 {
		c.Query.DaysAhead = 5
	}
	if c.Prometheus.Path == "" {
		c.Prometheus.Path = "/metrics"
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
}

// GetQueryInterval returns the query interval as a duration
func (c *QueryConfig) GetQueryInterval() time.Duration {
	return time.Duration(c.Interval) * time.Second
}

// GetDatesToQuery returns the dates to query based on date range or DaysAhead
func (c *QueryConfig) GetDatesToQuery() []time.Time {
	// If start_date and end_date are specified, use date range
	if c.StartDate != "" && c.EndDate != "" {
		startDate, err := time.Parse("2006-01-02", c.StartDate)
		if err != nil {
			fmt.Printf("Warning: Invalid start_date format: %v, falling back to days_ahead\n", err)
			return c.getDatesFromDaysAhead()
		}
		endDate, err := time.Parse("2006-01-02", c.EndDate)
		if err != nil {
			fmt.Printf("Warning: Invalid end_date format: %v, falling back to days_ahead\n", err)
			return c.getDatesFromDaysAhead()
		}
		if endDate.Before(startDate) {
			fmt.Println("Warning: end_date is before start_date, falling back to days_ahead")
			return c.getDatesFromDaysAhead()
		}

		var dates []time.Time
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dates = append(dates, d)
		}
		return dates
	}

	return c.getDatesFromDaysAhead()
}

func (c *QueryConfig) getDatesFromDaysAhead() []time.Time {
	dates := make([]time.Time, c.DaysAhead)
	now := time.Now()
	for i := 0; i < c.DaysAhead; i++ {
		dates[i] = now.AddDate(0, 0, i+1)
	}
	return dates
}
