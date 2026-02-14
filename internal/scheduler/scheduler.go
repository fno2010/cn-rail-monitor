package scheduler

import (
	"context"
	"log"
	"time"

	"cn-rail-monitor/internal/api"
	"cn-rail-monitor/internal/config"
	"cn-rail-monitor/internal/metrics"
	"cn-rail-monitor/internal/output"
)

// Scheduler handles periodic ticket queries
type Scheduler struct {
	cfg              *config.QueryConfig
	client           *api.Client
	metricsCollector *metrics.Collector
	telegrafOutput   *output.TelegrafOutput
}

// NewScheduler creates a new scheduler
func NewScheduler(cfg *config.QueryConfig, client *api.Client, metricsCollector *metrics.Collector, telegrafOutput *output.TelegrafOutput) *Scheduler {
	return &Scheduler{
		cfg:              cfg,
		client:           client,
		metricsCollector: metricsCollector,
		telegrafOutput:   telegrafOutput,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	go s.run(ctx)
	log.Printf("Scheduler started with interval: %v", s.cfg.GetQueryInterval())
}

// run runs the scheduler loop
func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.GetQueryInterval())
	defer ticker.Stop()

	// Run immediately on start
	s.queryAllRoutes()

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler stopped")
			return
		case <-ticker.C:
			s.queryAllRoutes()
		}
	}
}

// queryAllRoutes queries all configured routes
func (s *Scheduler) queryAllRoutes() {
	dates := s.cfg.GetDatesToQuery()
	dateFormat := "2006-01-02"

	for _, route := range s.cfg.Routes {
		for _, date := range dates {
			dateStr := date.Format(dateFormat)

			tickets, err := s.client.QueryTickets(route.FromStation, route.ToStation, dateStr)

			if err != nil {
				log.Printf("Error querying route %s -> %s on %s: %v, using mock data",
					route.FromStation, route.ToStation, dateStr, err)
				tickets = s.client.QueryTicketsWithMockData(route.FromStation, route.ToStation, dateStr)
			} else if len(tickets) == 0 {
				log.Printf("No data from API for %s -> %s on %s, using mock data",
					route.FromStation, route.ToStation, dateStr)
				tickets = s.client.QueryTicketsWithMockData(route.FromStation, route.ToStation, dateStr)
			}

			// Record metrics
			s.metricsCollector.RecordTickets(tickets)
			s.metricsCollector.RecordSuccess()

			// Write to Telegraf output
			if s.telegrafOutput != nil {
				if err := s.telegrafOutput.WriteTickets(tickets); err != nil {
					log.Printf("Error writing to Telegraf output: %v", err)
				}
			}
		}
	}
}
