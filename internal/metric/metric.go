package metric

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	StatusSuccess  = "success"
	StatusError    = "error"
	StatusCanceled = "canceled"
	StatusTimeout  = "timeout"
)

type Monitoring interface {
	IncSuccess(operation string)
	IncError(operation string)
	IncCanceled(operation string)
	IncTimeout(operation string)
	Observe(operation string, start time.Time)
}

type Metrics struct {
	Counter  *prometheus.CounterVec
	Duration *prometheus.HistogramVec
}

func New(name string) *Metrics {
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name + "_operations_total",
		Help: "Total count of " + name + " operations",
	},
		[]string{"operation", "status"},
	)

	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name + "_operation_duration_seconds",
		Help:    "Duration of " + name + " operations",
		Buckets: prometheus.DefBuckets,
	},
		[]string{"operation"},
	)

	prometheus.MustRegister(counter, duration)

	return &Metrics{
		Counter:  counter,
		Duration: duration,
	}
}

func (m *Metrics) IncSuccess(operation string) {
	m.Counter.WithLabelValues(operation, StatusSuccess).Inc()
}

func (m *Metrics) IncError(operation string) {
	m.Counter.WithLabelValues(operation, StatusError).Inc()
}

func (m *Metrics) IncCanceled(operation string) {
	m.Counter.WithLabelValues(operation, StatusCanceled).Inc()
}

func (m *Metrics) IncTimeout(operation string) {
	m.Counter.WithLabelValues(operation, StatusTimeout).Inc()
}

func (m *Metrics) Observe(operation string, start time.Time) {
	duration := time.Since(start).Seconds()
	m.Duration.WithLabelValues(operation).Observe(duration)
}

type NopMetrics struct{}

func NewNop() *NopMetrics {
	return &NopMetrics{}
}

func (nm *NopMetrics) IncSuccess(string) {}

func (nm *NopMetrics) IncError(string) {}

func (nm *NopMetrics) IncCanceled(string) {}

func (nm *NopMetrics) IncTimeout(string) {}

func (nm *NopMetrics) Observe(string, time.Time) {}
