package output

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cn-rail-monitor/internal/api"
	"cn-rail-monitor/internal/config"
)

// TelegrafOutput handles Telegraf-compatible output
type TelegrafOutput struct {
	cfg        *config.TelegrafConfig
	fileHandle *os.File
}

// NewTelegrafOutput creates a new Telegraf output handler
func NewTelegrafOutput(cfg *config.TelegrafConfig) (*TelegrafOutput, error) {
	if !cfg.Enabled {
		return &TelegrafOutput{cfg: cfg}, nil
	}

	var fileHandle *os.File
	if cfg.OutputMode == "file" && cfg.OutputPath != "" {
		f, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open output file: %w", err)
		}
		fileHandle = f
		log.Printf("Telegraf output enabled: writing to %s", cfg.OutputPath)
	} else {
		log.Println("Telegraf output enabled: writing to stdout")
	}

	return &TelegrafOutput{
		cfg:        cfg,
		fileHandle: fileHandle,
	}, nil
}

// WriteTickets writes ticket data in InfluxDB line protocol format
func (t *TelegrafOutput) WriteTickets(tickets []api.TicketInfo) error {
	if !t.cfg.Enabled {
		return nil
	}

	var sb strings.Builder
	timestamp := time.Now().UnixNano()

	for _, ticket := range tickets {
		// Line protocol format: measurement,tags values timestamp
		line := fmt.Sprintf("train_tickets,train_no=%s,train_type=%s,from_station=%s,to_station=%s,date=%s,seat_type=%s available=%d,price=%.2f %d",
			sanitizeTag(ticket.TrainNo),
			sanitizeTag(ticket.TrainType),
			sanitizeTag(ticket.FromStation),
			sanitizeTag(ticket.ToStation),
			sanitizeTag(ticket.Date),
			sanitizeTag(ticket.SeatType),
			ticket.Available,
			ticket.Price,
			timestamp,
		)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	output := sb.String()
	if output == "" {
		return nil
	}

	if t.cfg.OutputMode == "file" && t.fileHandle != nil {
		_, err := t.fileHandle.WriteString(output)
		if err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	} else {
		fmt.Println(output)
	}

	return nil
}

// sanitizeTag sanitizes a string for use as a tag value in InfluxDB line protocol
func sanitizeTag(s string) string {
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "=", "\\=")
	return s
}

// Close closes the output file
func (t *TelegrafOutput) Close() error {
	if t.fileHandle != nil {
		return t.fileHandle.Close()
	}
	return nil
}
