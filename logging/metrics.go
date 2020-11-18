package logging

import (
	"fmt"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/statsd"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics interface {
	Timing(name string, duration time.Duration, tags []string)
	Increment(name string, tags []string)
}

type NullMetrics struct {
}

func (s *NullMetrics) Timing(name string, duration time.Duration, tags []string) {}
func (s *NullMetrics) Increment(name string, tags []string)                      {}

type StatsDMetrics struct {
	c *statsd.Client
}

func NewStatsDMetrics(serviceName, environment, uri string) Metrics {
	c, _ := statsd.New(uri)
	c.Tags = []string{
		fmt.Sprintf("service:%s", serviceName),
		fmt.Sprintf("env:%s", environment),
	}

	return &StatsDMetrics{
		c: c,
	}
}

func (s *StatsDMetrics) Timing(name string, duration time.Duration, tags []string) {
	s.c.Timing(name, duration, tags, 1)
}

func (s *StatsDMetrics) Increment(name string, tags []string) {
	s.c.Incr(name, tags, 1)
}

type PrometheusMetrics struct {
	counters map[string]prometheus.Counter
	timers   map[string]prometheus.Histogram
}

func NewPrometheusMetrics() Metrics {
	return &PrometheusMetrics{
		counters: make(map[string]prometheus.Counter),
		timers:   make(map[string]prometheus.Histogram),
	}
}

func (p *PrometheusMetrics) Timing(name string, duration time.Duration, tags []string) {
	name = strings.ReplaceAll(name, ".", "_") + "_seconds"
	timer, ok := p.timers[name]
	if !ok {
		timer = promauto.NewHistogram(prometheus.HistogramOpts{
			Name: name,
			Help: name,
			// []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
			//Buckets: []float64{0.0001, 0.00025, 0.0005, 0.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			Buckets: prometheus.ExponentialBuckets((1 * time.Microsecond).Seconds(), 10, 7),
		})
		p.timers[name] = timer
	}

	timer.Observe(duration.Seconds())
}

func (p *PrometheusMetrics) Increment(name string, tags []string) {
	// todo: add tags
	name = strings.ReplaceAll(name, ".", "_") + "_total"
	counter, ok := p.counters[name]
	if !ok {
		counter = promauto.NewCounter(prometheus.CounterOpts{
			Name: name,
			Help: name,
		})
		p.counters[name] = counter
	}

	counter.Inc()
}
