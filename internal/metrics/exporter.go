package metrics

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"cn-rail-monitor/internal/api"
	"cn-rail-monitor/internal/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector handles metrics collection
type Collector struct {
	cfg *config.Config

	// Prometheus metrics
	queryTotal       prometheus.Counter
	queryErrorsTotal prometheus.Counter
	availableSeats   *prometheus.GaugeVec
	ticketPrice      *prometheus.GaugeVec

	mu sync.RWMutex
	// Cache for latest ticket data
	latestTickets []api.TicketInfo
}

// NewCollector creates a new metrics collector
func NewCollector(cfg *config.Config) *Collector {
	c := &Collector{
		cfg: cfg,
		queryTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "train_ticket_query_total",
			Help: "Total number of ticket queries made",
		}),
		queryErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "train_ticket_query_errors_total",
			Help: "Total number of ticket query errors",
		}),
		availableSeats: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "train_ticket_available_seats",
			Help: "Available seat count per train/date/seat_type",
		}, []string{"train_no", "train_type", "from_station", "to_station", "date", "seat_type"}),
		ticketPrice: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "train_ticket_price",
			Help: "Price per train/date/seat_type in CNY",
		}, []string{"train_no", "train_type", "from_station", "to_station", "date", "seat_type"}),
	}

	return c
}

// RecordTickets records ticket data and updates metrics
func (c *Collector) RecordTickets(tickets []api.TicketInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.latestTickets = tickets

	// Clear previous metrics
	c.availableSeats.Reset()
	c.ticketPrice.Reset()

	// Update metrics with new data
	for _, ticket := range tickets {
		labels := prometheus.Labels{
			"train_no":     ticket.TrainNo,
			"train_type":   ticket.TrainType,
			"from_station": ticket.FromStation,
			"to_station":   ticket.ToStation,
			"date":         ticket.Date,
			"seat_type":    ticket.SeatType,
		}

		c.availableSeats.With(labels).Set(float64(ticket.Available))
		c.ticketPrice.With(labels).Set(ticket.Price)
	}

	log.Printf("Recorded %d ticket records with metrics", len(tickets))
}

// RecordSuccess increments the success counter
func (c *Collector) RecordSuccess() {
	c.queryTotal.Inc()
}

// RecordError increments the error counter
func (c *Collector) RecordError() {
	c.queryErrorsTotal.Inc()
}

// GetLatestTickets returns the latest ticket data
func (c *Collector) GetLatestTickets() []api.TicketInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latestTickets
}

// DebugPrint prints current metrics to the writer
func (c *Collector) DebugPrint(w io.Writer) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fmt.Fprintf(w, "# Debug: Latest Tickets (count: %d)\n", len(c.latestTickets))
	for i, ticket := range c.latestTickets {
		fmt.Fprintf(w, "[%d] %s %s -> %s | Date: %s | Seat: %s | Available: %d | Price: %.2f\n",
			i, ticket.TrainNo, ticket.FromStation, ticket.ToStation,
			ticket.Date, ticket.SeatType, ticket.Available, ticket.Price)
	}
}

// DebugPrintHandler returns an HTTP handler for debugging
func DebugPrintHandler(collector *Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		collector.DebugPrint(w)
	}
}
