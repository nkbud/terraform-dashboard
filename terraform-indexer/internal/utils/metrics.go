package utils

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all the Prometheus metrics for the application
type Metrics struct {
	FilesCollected    prometheus.CounterVec
	ObjectsParsed     prometheus.CounterVec
	ObjectsWritten    prometheus.CounterVec
	ProcessingErrors  prometheus.CounterVec
	ProcessingTime    prometheus.HistogramVec
	QueueSize         prometheus.GaugeVec
	WorkerStatus      prometheus.GaugeVec
}

// NewMetrics creates and registers Prometheus metrics
func NewMetrics() *Metrics {
	return &Metrics{
		FilesCollected: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "terraform_indexer_files_collected_total",
				Help: "Total number of Terraform files collected",
			},
			[]string{"source", "file_type"},
		),
		ObjectsParsed: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "terraform_indexer_objects_parsed_total",
				Help: "Total number of Terraform objects parsed",
			},
			[]string{"object_type", "file_type"},
		),
		ObjectsWritten: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "terraform_indexer_objects_written_total",
				Help: "Total number of Terraform objects written to database",
			},
			[]string{"object_type"},
		),
		ProcessingErrors: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "terraform_indexer_processing_errors_total",
				Help: "Total number of processing errors",
			},
			[]string{"component", "error_type"},
		),
		ProcessingTime: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "terraform_indexer_processing_duration_seconds",
				Help:    "Time spent processing files and objects",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"component", "operation"},
		),
		QueueSize: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "terraform_indexer_queue_size",
				Help: "Current size of processing queues",
			},
			[]string{"queue_type"},
		),
		WorkerStatus: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "terraform_indexer_worker_status",
				Help: "Status of worker processes (1=running, 0=stopped)",
			},
			[]string{"worker_type", "worker_id"},
		),
	}
}