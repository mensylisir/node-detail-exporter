package collector

import (
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Collector is the interface a collector has to implement.
type Collector interface {
	// Update fetches the data from the source and updates the Prometheus metrics.
	// It is called periodically in a background goroutine.
	Update(ch chan<- prometheus.Metric, interval time.Duration) error
}

// Registry manages a set of collectors.
type Registry struct {
	mu         sync.Mutex
	collectors map[string]Collector
	metrics    []prometheus.Metric
}

// NewRegistry creates a new collector registry.
func NewRegistry() *Registry {
	return &Registry{
		collectors: make(map[string]Collector),
	}
}

// Register registers a new collector.
func (r *Registry) Register(name string, collector Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors[name] = collector
}

// UpdateAll updates all registered collectors.
func (r *Registry) UpdateAll(interval time.Duration) {
	var wg sync.WaitGroup
	ch := make(chan prometheus.Metric)

	for name, c := range r.collectors {
		wg.Add(1)
		go func(name string, c Collector) {
			defer wg.Done()
			if err := c.Update(ch, interval); err != nil {
				log.Printf("Error updating collector %s: %v", name, err)
			}
		}(name, c)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics = make([]prometheus.Metric, 0)
	for metric := range ch {
		r.metrics = append(r.metrics, metric)
	}
}

func (r *Registry) Describe(ch chan<- *prometheus.Desc) {
	// Since collectors are dynamic, we can't describe them here.
	// We will just send a dummy descriptor.
	ch <- prometheus.NewDesc("dummy", "dummy", nil, nil)
}

func (r *Registry) Collect(ch chan<- prometheus.Metric) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, metric := range r.metrics {
		ch <- metric
	}
}
