package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cn-rail-monitor/internal/api"
	"cn-rail-monitor/internal/config"
)

func TestSanitizeTag(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"正常文本", "正常文本"},
		{"hello,world", "hello\\,world"},
		{"hello world", "hello_world"},
		{"key=value", "key\\=value"},
		{"a,b=c d", "a\\,b\\=c_d"},
		{"", ""},
		{"   ", "___"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeTag(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeTag(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWriteTicketsDisabled(t *testing.T) {
	cfg := &config.TelegrafConfig{
		Enabled: false,
	}

	out, err := NewTelegrafOutput(cfg)
	if err != nil {
		t.Fatalf("NewTelegrafOutput error = %v", err)
	}

	tickets := []api.TicketInfo{
		{TrainInfo: api.TrainInfo{TrainNo: "G531"}, Available: 10},
	}

	err = out.WriteTickets(tickets)
	if err != nil {
		t.Errorf("WriteTickets error = %v", err)
	}
}

func TestWriteTicketsToStdout(t *testing.T) {
	cfg := &config.TelegrafConfig{
		Enabled:    true,
		OutputMode: "stdout",
	}

	out, err := NewTelegrafOutput(cfg)
	if err != nil {
		t.Fatalf("NewTelegrafOutput error = %v", err)
	}

	tickets := []api.TicketInfo{
		{
			TrainInfo: api.TrainInfo{
				TrainNo:     "G531",
				TrainType:   "G",
				FromStation: "北京南",
				ToStation:   "上海虹桥",
			},
			Date:      "2026-02-20",
			SeatType:  "二等座",
			Available: 100,
			Price:     553.0,
		},
	}

	err = out.WriteTickets(tickets)
	if err != nil {
		t.Errorf("WriteTickets error = %v", err)
	}
}

func TestWriteTicketsToFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "telegraf_test.log")

	cfg := &config.TelegrafConfig{
		Enabled:    true,
		OutputMode: "file",
		OutputPath: tmpFile,
	}

	out, err := NewTelegrafOutput(cfg)
	if err != nil {
		t.Fatalf("NewTelegrafOutput error = %v", err)
	}
	defer out.Close()

	tickets := []api.TicketInfo{
		{
			TrainInfo: api.TrainInfo{
				TrainNo:     "G531",
				TrainType:   "G",
				FromStation: "北京南",
				ToStation:   "上海虹桥",
			},
			Date:      "2026-02-20",
			SeatType:  "二等座",
			Available: 100,
			Price:     553.0,
		},
		{
			TrainInfo: api.TrainInfo{
				TrainNo:     "G532",
				TrainType:   "G",
				FromStation: "北京南",
				ToStation:   "上海虹桥",
			},
			Date:      "2026-02-20",
			SeatType:  "一等座",
			Available: 50,
			Price:     933.5,
		},
	}

	err = out.WriteTickets(tickets)
	if err != nil {
		t.Errorf("WriteTickets error = %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Error("output file is empty")
	}

	if !strings.Contains(content, "G531") {
		t.Error("output missing G531")
	}
	if !strings.Contains(content, "G532") {
		t.Error("output missing G532")
	}
}

func TestWriteTicketsEmpty(t *testing.T) {
	cfg := &config.TelegrafConfig{
		Enabled:    true,
		OutputMode: "stdout",
	}

	out, err := NewTelegrafOutput(cfg)
	if err != nil {
		t.Fatalf("NewTelegrafOutput error = %v", err)
	}

	err = out.WriteTickets(nil)
	if err != nil {
		t.Errorf("WriteTickets error = %v", err)
	}

	err = out.WriteTickets([]api.TicketInfo{})
	if err != nil {
		t.Errorf("WriteTickets error = %v", err)
	}
}

func TestWriteTicketsWithSpecialChars(t *testing.T) {
	cfg := &config.TelegrafConfig{
		Enabled:    true,
		OutputMode: "stdout",
	}

	out, err := NewTelegrafOutput(cfg)
	if err != nil {
		t.Fatalf("NewTelegrafOutput error = %v", err)
	}

	tickets := []api.TicketInfo{
		{
			TrainInfo: api.TrainInfo{
				TrainNo:     "G531",
				FromStation: "北京,北",
				ToStation:   "上海=虹桥",
			},
			SeatType: "一等座",
		},
	}

	err = out.WriteTickets(tickets)
	if err != nil {
		t.Errorf("WriteTickets error = %v", err)
	}
}

func TestTelegrafOutputClose(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "close_test.log")

	cfg := &config.TelegrafConfig{
		Enabled:    true,
		OutputMode: "file",
		OutputPath: tmpFile,
	}

	out, err := NewTelegrafOutput(cfg)
	if err != nil {
		t.Fatalf("NewTelegrafOutput error = %v", err)
	}

	err = out.Close()
	if err != nil {
		t.Errorf("Close error = %v", err)
	}
}

func TestNewTelegrafOutputInvalidPath(t *testing.T) {
	cfg := &config.TelegrafConfig{
		Enabled:    true,
		OutputMode: "file",
		OutputPath: "/nonexistent/path/that/cannot/be/created/file.log",
	}

	_, err := NewTelegrafOutput(cfg)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestWriteTicketsFormat(t *testing.T) {
	cfg := &config.TelegrafConfig{
		Enabled:    true,
		OutputMode: "stdout",
	}

	out, err := NewTelegrafOutput(cfg)
	if err != nil {
		t.Fatalf("NewTelegrafOutput error = %v", err)
	}

	tickets := []api.TicketInfo{
		{
			TrainInfo: api.TrainInfo{
				TrainNo:     "G123",
				TrainType:   "G",
				FromStation: "北京",
				ToStation:   "上海",
			},
			Date:      "2026-01-01",
			SeatType:  "二等座",
			Available: 99,
			Price:     550.0,
		},
	}

	err = out.WriteTickets(tickets)
	if err != nil {
		t.Errorf("WriteTickets error = %v", err)
	}
}
