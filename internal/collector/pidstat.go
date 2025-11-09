package collector

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/process"
)

var (
	processCPUUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_process_cpu_usage_percent",
			Help: "Per-process CPU usage from pidstat.",
		},
		[]string{"pid", "user", "command"},
	)
	processIOWrite = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_process_io_write_kilobytes_per_second",
			Help: "Per-process disk write throughput from pidstat.",
		},
		[]string{"pid", "user", "command"},
	)
	processIORead = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_process_io_read_kilobytes_per_second",
			Help: "Per-process disk read throughput from pidstat.",
		},
		[]string{"pid", "user", "command"},
	)
	processContextSwitchesVoluntary = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_process_context_switches_voluntary_per_second",
			Help: "Per-process voluntary context switches from pidstat.",
		},
		[]string{"pid", "user", "command"},
	)
	processContextSwitchesNonVoluntary = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_process_context_switches_non_voluntary_per_second",
			Help: "Per-process non-voluntary context switches from pidstat.",
		},
		[]string{"pid", "user", "command"},
	)
)

type PidstatCollector struct {
	previousProcs map[int32]*process.IOCountersStat
	previousCtx   map[int32]*process.NumCtxSwitchesStat
	mu            sync.Mutex
}

func NewPidstatCollector() (Collector, error) {
	return &PidstatCollector{
		previousProcs: make(map[int32]*process.IOCountersStat),
		previousCtx:   make(map[int32]*process.NumCtxSwitchesStat),
	}, nil
}

func (c *PidstatCollector) Name() string {
	return "pidstat"
}

func (c *PidstatCollector) Update(ch chan<- prometheus.Metric, interval time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	processCPUUsage.Reset()
	processIOWrite.Reset()
	processIORead.Reset()
	processContextSwitchesVoluntary.Reset()
	processContextSwitchesNonVoluntary.Reset()

	processes, err := process.Processes()
	if err != nil {
		log.Printf("Error getting processes: %v", err)
		return err
	}

	currentProcs := make(map[int32]*process.IOCountersStat)
	currentCtx := make(map[int32]*process.NumCtxSwitchesStat)

	var wg sync.WaitGroup
	for _, p := range processes {
		wg.Add(1)
		go func(p *process.Process) {
			defer wg.Done()
			c.collectProcessMetrics(p, interval, currentProcs, currentCtx)
		}(p)
	}
	wg.Wait()

	c.previousProcs = currentProcs
	c.previousCtx = currentCtx

	processCPUUsage.Collect(ch)
	processIOWrite.Collect(ch)
	processIORead.Collect(ch)
	processContextSwitchesVoluntary.Collect(ch)
	processContextSwitchesNonVoluntary.Collect(ch)

	return nil
}

func (c *PidstatCollector) collectProcessMetrics(p *process.Process, interval time.Duration,
	currentProcs map[int32]*process.IOCountersStat, currentCtx map[int32]*process.NumCtxSwitchesStat) {
	pid := p.Pid
	user, err := p.Username()
	if err != nil {
		return
	}
	command, err := p.Name()
	if err != nil {
		return
	}
	pidStr := strconv.Itoa(int(pid))

	// CPU Usage
	cpu, err := p.CPUPercent()
	if err == nil {
		processCPUUsage.WithLabelValues(pidStr, user, command).Set(cpu)
	}

	// IO Counters
	io, err := p.IOCounters()
	if err == nil {
		if prevIO, ok := c.previousProcs[pid]; ok {
			readRate := float64(io.ReadBytes-prevIO.ReadBytes) / interval.Seconds() / 1024
			writeRate := float64(io.WriteBytes-prevIO.WriteBytes) / interval.Seconds() / 1024
			processIORead.WithLabelValues(pidStr, user, command).Set(readRate)
			processIOWrite.WithLabelValues(pidStr, user, command).Set(writeRate)
		}
		currentProcs[pid] = io
	}

	// Context Switches
	ctx, err := p.NumCtxSwitches()
	if err == nil {
		if prevCtx, ok := c.previousCtx[pid]; ok {
			voluntaryRate := float64(ctx.Voluntary-prevCtx.Voluntary) / interval.Seconds()
			involuntaryRate := float64(ctx.Involuntary-prevCtx.Involuntary) / interval.Seconds()
			processContextSwitchesVoluntary.WithLabelValues(pidStr, user, command).Set(voluntaryRate)
			processContextSwitchesNonVoluntary.WithLabelValues(pidStr, user, command).Set(involuntaryRate)
		}
		currentCtx[pid] = ctx
	}
}
