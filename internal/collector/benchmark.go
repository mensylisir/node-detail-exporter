package collector

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	benchmarkFsyncP99 = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "node_benchmark_fsync_p99_seconds",
			Help: "Benchmark result: 99th percentile of single-thread fsync latency.",
		},
	)
	benchmarkCPUEventsPerSecond = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "node_benchmark_cpu_events_per_second",
			Help: "Benchmark result: CPU events per second.",
		},
	)
	benchmarkMemoryOpsPerSecond = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "node_benchmark_memory_ops_per_second",
			Help: "Benchmark result: Memory operations per second.",
		},
	)
	benchmarkNetBandwidthBitsPerSecond = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "node_benchmark_net_bandwidth_bits_per_second",
			Help: "Benchmark result: Network bandwidth in bits per second.",
		},
	)
)

type BenchmarkCollector struct {
	sync.Once
}

func NewBenchmarkCollector() (Collector, error) {
	return &BenchmarkCollector{}, nil
}

func (c *BenchmarkCollector) Update(ch chan<- prometheus.Metric, interval time.Duration) error {
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
		if cpu, ok := benchmarks["cpu_events_per_second"]; ok {
			benchmarkCPUEventsPerSecond.Set(cpu)
			log.Printf("Benchmark cpu_events_per_second loaded: %f", cpu)
		}
		if mem, ok := benchmarks["memory_ops_per_second"]; ok {
			benchmarkMemoryOpsPerSecond.Set(mem)
			log.Printf("Benchmark memory_ops_per_second loaded: %f", mem)
		}
		if net, ok := benchmarks["net_bandwidth_bits_per_second"]; ok {
			benchmarkNetBandwidthBitsPerSecond.Set(net)
			log.Printf("Benchmark net_bandwidth_bits_per_second loaded: %f", net)
		}
	})

	benchmarkFsyncP99.Collect(ch)
	benchmarkCPUEventsPerSecond.Collect(ch)
	benchmarkMemoryOpsPerSecond.Collect(ch)
	benchmarkNetBandwidthBitsPerSecond.Collect(ch)
	return nil
}
