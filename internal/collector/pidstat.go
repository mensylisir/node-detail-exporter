package collector

import (
	"bufio"
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"node-prober/internal/parser"
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

type PidstatCollector struct{}

func NewPidstatCollector() (Collector, error) {
	return &PidstatCollector{}, nil
}

func (c *PidstatCollector) Update(ch chan<- prometheus.Metric) error {
	processCPUUsage.Reset()
	processIOWrite.Reset()
	processIORead.Reset()
	processContextSwitchesVoluntary.Reset()
	processContextSwitchesNonVoluntary.Reset()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		collectFromCommand("pidstat", []string{"-u", "1", "1"}, parsePidstatCPU)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		collectFromCommand("pidstat", []string{"-d", "1", "1"}, parsePidstatIO)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		collectFromCommand("pidstat", []string{"-w", "1", "1"}, parsePidstatContextSwitches)
	}()

	wg.Wait()

	processCPUUsage.Collect(ch)
	processIOWrite.Collect(ch)
	processIORead.Collect(ch)
	processContextSwitchesVoluntary.Collect(ch)
	processContextSwitchesNonVoluntary.Collect(ch)

	return nil
}

func collectFromCommand(command string, args []string, parser func(io.Reader)) {
	cmd := exec.Command(command, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error creating stdout pipe for %s: %v", command, err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Error starting %s: %v", command, err)
		return
	}

	parser(stdout)

	cmd.Wait()
}

func parsePidstatCPU(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 8 || fields[0] == "#" || !parser.IsNumeric(fields[2]) {
			continue
		}

		pid := fields[2]
		user := fields[1]
		cpu, err := strconv.ParseFloat(fields[7], 64)
		if err != nil {
			log.Printf("Error parsing CPU value in pidstat: %v", err)
			continue
		}
		command := fields[len(fields)-1]

		processCPUUsage.WithLabelValues(pid, user, command).Set(cpu)
	}
}

func parsePidstatIO(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 6 || fields[0] == "#" || !parser.IsNumeric(fields[2]) {
			continue
		}

		pid := fields[2]
		user := fields[1]
		readKB, err := strconv.ParseFloat(fields[3], 64)
		if err != nil {
			log.Printf("Error parsing readKB value in pidstat: %v", err)
			continue
		}
		writeKB, err := strconv.ParseFloat(fields[4], 64)
		if err != nil {
			log.Printf("Error parsing writeKB value in pidstat: %v", err)
			continue
		}
		command := fields[len(fields)-1]

		processIORead.WithLabelValues(pid, user, command).Set(readKB)
		processIOWrite.WithLabelValues(pid, user, command).Set(writeKB)
	}
}

func parsePidstatContextSwitches(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 6 || fields[0] == "#" || !parser.IsNumeric(fields[2]) {
			continue
		}

		pid := fields[2]
		user := fields[1]
		voluntary, err := strconv.ParseFloat(fields[3], 64)
		if err != nil {
			log.Printf("Error parsing voluntary context switches in pidstat: %v", err)
			continue
		}
		nonVoluntary, err := strconv.ParseFloat(fields[4], 64)
		if err != nil {
			log.Printf("Error parsing non-voluntary context switches in pidstat: %v", err)
			continue
		}
		command := fields[len(fields)-1]

		processContextSwitchesVoluntary.WithLabelValues(pid, user, command).Set(voluntary)
		processContextSwitchesNonVoluntary.WithLabelValues(pid, user, command).Set(nonVoluntary)
	}
}
