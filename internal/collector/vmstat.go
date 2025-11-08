package collector

import (
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"
)

var (
	vmstatProcsRunning = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "node_vmstat_procs_running",
			Help: "Number of running processes.",
		},
	)
	vmstatProcsBlocked = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "node_vmstat_procs_blocked",
			Help: "Number of blocked processes.",
		},
	)
	vmstatCPUStealPercent = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "node_vmstat_cpu_steal_percent",
			Help: "Stolen time from a virtual machine.",
		},
	)
	vmstatIOWaitPercent = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "node_vmstat_io_wait_percent",
			Help: "Time spent waiting for I/O.",
		},
	)
)

type VmstatCollector struct {
	prevCPUTimes *cpu.TimesStat
	mu           sync.Mutex
}

func NewVmstatCollector() (Collector, error) {
	return &VmstatCollector{}, nil
}

func (c *VmstatCollector) Update(ch chan<- prometheus.Metric, interval time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// CPU metrics
	cpuTimes, err := cpu.Times(false)
	if err != nil {
		log.Printf("Error getting cpu times: %v", err)
	} else if len(cpuTimes) > 0 {
		if c.prevCPUTimes != nil {
			totalDelta := cpuTimes[0].Total() - c.prevCPUTimes.Total()
			if totalDelta > 0 {
				stealPercent := (cpuTimes[0].Steal - c.prevCPUTimes.Steal) / totalDelta * 100
				iowaitPercent := (cpuTimes[0].Iowait - c.prevCPUTimes.Iowait) / totalDelta * 100
				vmstatCPUStealPercent.Set(stealPercent)
				vmstatIOWaitPercent.Set(iowaitPercent)
			}
		}
		c.prevCPUTimes = &cpuTimes[0]
	}

	// Process metrics
	processes, err := process.Processes()
	if err != nil {
		log.Printf("Error getting processes: %v", err)
	} else {
		var running, blocked int64
		for _, p := range processes {
			status, err := p.Status()
			if err != nil {
				continue
			}
			s := status
			if len(s) > 0 {
				switch s[0] {
				case 'R':
					running++
				case 'S', 'D', 'T':
					blocked++
				}
			}
		}
		vmstatProcsRunning.Set(float64(running))
		vmstatProcsBlocked.Set(float64(blocked))
	}

	ch <- vmstatProcsRunning
	ch <- vmstatProcsBlocked
	ch <- vmstatCPUStealPercent
	ch <- vmstatIOWaitPercent

	return nil
}
