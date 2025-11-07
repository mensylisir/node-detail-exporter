package collector

import (
	"bufio"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"node-prober/internal/parser"
)

var (
	vmstatProcsRunning = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "node_vmstat_procs_running",
			Help: "Number of running processes (procs r from vmstat).",
		},
	)
	vmstatProcsBlocked = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "node_vmstat_procs_blocked",
			Help: "Number of blocked processes (procs b from vmstat).",
		},
	)
	vmstatCPUStealPercent = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "node_vmstat_cpu_steal_percent",
			Help: "Stolen time from a virtual machine (cpu st from vmstat).",
		},
	)
	vmstatIOWaitPercent = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "node_vmstat_io_wait_percent",
			Help: "Time spent waiting for I/O (cpu wa from vmstat).",
		},
	)
)

type VmstatCollector struct{}

func NewVmstatCollector() (Collector, error) {
	return &VmstatCollector{}, nil
}

func (c *VmstatCollector) Update(ch chan<- prometheus.Metric) error {
	parser.CollectFromCommand("vmstat", []string{"-n", "1", "2"}, parseVmstat)

	ch <- vmstatProcsRunning
	ch <- vmstatProcsBlocked
	ch <- vmstatCPUStealPercent
	ch <- vmstatIOWaitPercent

	return nil
}

func parseVmstat(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	lineCount := 0
	dataLine := ""
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		if lineCount == 3 {
			dataLine = line
			break
		}
	}

	if dataLine == "" {
		log.Println("Could not find vmstat data line")
		return
	}

	fields := strings.Fields(dataLine)
	if len(fields) < 17 {
		log.Printf("Unexpected number of fields in vmstat output: got %d, want >= 17", len(fields))
		return
	}

	r, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		log.Printf("Error parsing running procs in vmstat: %v", err)
	}
	b, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		log.Printf("Error parsing blocked procs in vmstat: %v", err)
	}
	wa, err := strconv.ParseFloat(fields[16], 64)
	if err != nil {
		log.Printf("Error parsing io wait in vmstat: %v", err)
	}
	st, err := strconv.ParseFloat(fields[17], 64)
	if err != nil {
		log.Printf("Error parsing steal time in vmstat: %v", err)
	}

	vmstatProcsRunning.Set(r)
	vmstatProcsBlocked.Set(b)
	vmstatIOWaitPercent.Set(wa)
	vmstatCPUStealPercent.Set(st)
}
