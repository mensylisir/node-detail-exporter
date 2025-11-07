package collector

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	benchmarkFsyncP99 = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "node_benchmark_fsync_p99_seconds",
			Help: "Benchmark result: 99th percentile of single-thread fsync latency.",
		},
	)
)

type BenchmarkCollector struct {
	sync.Once
}

func NewBenchmarkCollector() (Collector, error) {
	return &BenchmarkCollector{}, nil
}

func (c *BenchmarkCollector) Update(ch chan<- prometheus.Metric) error {
	c.Do(func() {
		log.Println("Loading benchmark data...")
		data, err := ioutil.ReadFile("configs/benchmarks.json")
		if err != nil {
			log.Printf("Error reading benchmark file: %v", err)
			return
		}

		var benchmarks map[string]float64
		if err := json.Unmarshal(data, &benchmarks); err != nil {
			log.Printf("Error parsing benchmark JSON: %v", err)
			return
		}

		if p99, ok := benchmarks["fsync_p99_seconds"]; ok {
			benchmarkFsyncP99.Set(p99)
			log.Printf("Benchmark fsync_p99_seconds loaded: %f", p99)
		}
	})

	benchmarkFsyncP99.Collect(ch)
	return nil
}
