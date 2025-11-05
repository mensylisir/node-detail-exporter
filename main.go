package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// 定义 Prometheus 指标
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

var (
	iostatAwait = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_iostat_await_milliseconds",
			Help: "Average time for I/O requests issued to the device to be served (await from iostat).",
		},
		[]string{"device"},
	)
	iostatReadAwait = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_iostat_read_await_milliseconds",
			Help: "Average time for read requests issued to the device to be served (r_await from iostat).",
		},
		[]string{"device"},
	)
	iostatWriteAwait = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_iostat_write_await_milliseconds",
			Help: "Average time for write requests issued to the device to be served (w_await from iostat).",
		},
		[]string{"device"},
	)
	iostatUtilPercent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_iostat_util_percent",
			Help: "Percentage of CPU time during which I/O requests were issued to the device (%util from iostat).",
		},
		[]string{"device"},
	)
)

// Pidstat Collector 结构体
type PidstatCollector struct {
	// ... 可以添加 iostat, vmstat 的 desc ...
}

// Vmstat Collector 结构体
type VmstatCollector struct{}

// Iostat Collector 结构体
type IostatCollector struct{}

func (c *PidstatCollector) Describe(ch chan<- *prometheus.Desc) {
	processCPUUsage.Describe(ch)
	processIOWrite.Describe(ch)
	processIORead.Describe(ch)
	processContextSwitchesVoluntary.Describe(ch)
	processContextSwitchesNonVoluntary.Describe(ch)
}

func (c *VmstatCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- vmstatProcsRunning.Desc()
	ch <- vmstatProcsBlocked.Desc()
	ch <- vmstatCPUStealPercent.Desc()
	ch <- vmstatIOWaitPercent.Desc()
}

func (c *PidstatCollector) Collect(ch chan<- prometheus.Metric) {
    // 重置 GaugeVec, 避免保留已退出进程的旧数据
	processCPUUsage.Reset()
	processIOWrite.Reset()
	processIORead.Reset()
	processContextSwitchesVoluntary.Reset()
	processContextSwitchesNonVoluntary.Reset()

	var wg sync.WaitGroup

	// --- 执行 pidstat -u (CPU) ---
	// -u: CPU, 1 1: 间隔1秒, 采样1次
	wg.Add(1)
	go func() {
		defer wg.Done()
		collectFromCommand("pidstat", []string{"-u", "1", "1"}, parsePidstatCPU)
	}()

	// --- 执行 pidstat -d (I/O) ---
	// -d: I/O, 1 1: 间隔1秒, 采样1次
	wg.Add(1)
	go func() {
		defer wg.Done()
		collectFromCommand("pidstat", []string{"-d", "1", "1"}, parsePidstatIO)
	}()

	// --- 执行 pidstat -w (Context Switches) ---
	// -w: context switches, 1 1: 间隔1秒, 采样1次
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


// 解析 pidstat -u 的输出
func parsePidstatCPU(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		// 跳过表头和空行
		if len(fields) < 8 || fields[0] == "#" || !isNumeric(fields[2]) {
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

// 解析 pidstat -d 的输出
func parsePidstatIO(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 6 || fields[0] == "#" || !isNumeric(fields[2]) {
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

// 解析 pidstat -w 的输出
func parsePidstatContextSwitches(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 6 || fields[0] == "#" || !isNumeric(fields[2]) {
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

func isNumeric(s string) bool {
    _, err := strconv.ParseFloat(s, 64)
    return err == nil
}

func (c *VmstatCollector) Collect(ch chan<- prometheus.Metric) {
	// -n: header only once, 1 2: 1s interval, 2 samples
	collectFromCommand("vmstat", []string{"-n", "1", "2"}, parseVmstat)

	ch <- vmstatProcsRunning
	ch <- vmstatProcsBlocked
	ch <- vmstatCPUStealPercent
	ch <- vmstatIOWaitPercent
}

func parseVmstat(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	lineCount := 0
	dataLine := ""
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		// The 3rd line contains the data we want (1st is header, 2nd is avg since boot)
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


func (c *IostatCollector) Describe(ch chan<- *prometheus.Desc) {
	iostatAwait.Describe(ch)
	iostatReadAwait.Describe(ch)
	iostatWriteAwait.Describe(ch)
	iostatUtilPercent.Describe(ch)
}

func (c *IostatCollector) Collect(ch chan<- prometheus.Metric) {
	iostatAwait.Reset()
	iostatReadAwait.Reset()
	iostatWriteAwait.Reset()
	iostatUtilPercent.Reset()

	// -x: extended stats, -d: device util, 1 1: 1s interval, 1 sample
	collectFromCommand("iostat", []string{"-x", "-d", "1", "1"}, parseIostat)

	iostatAwait.Collect(ch)
	iostatReadAwait.Collect(ch)
	iostatWriteAwait.Collect(ch)
	iostatUtilPercent.Collect(ch)
}

func parseIostat(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	headerFound := false
	for scanner.Scan() {
		line := scanner.Text()
		if !headerFound {
			if strings.HasPrefix(line, "Device") {
				headerFound = true
			}
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}

		device := fields[0]
		rAwait, err := strconv.ParseFloat(fields[10], 64)
		if err != nil {
			log.Printf("Error parsing r_await in iostat for device %s: %v", device, err)
			continue
		}
		wAwait, err := strconv.ParseFloat(fields[11], 64)
		if err != nil {
			log.Printf("Error parsing w_await in iostat for device %s: %v", device, err)
			continue
		}
		await, err := strconv.ParseFloat(fields[12], 64)
		if err != nil {
			log.Printf("Error parsing await in iostat for device %s: %v", device, err)
			continue
		}
		util, err := strconv.ParseFloat(fields[13], 64)
		if err != nil {
			log.Printf("Error parsing %%util in iostat for device %s: %v", device, err)
			continue
		}

		iostatReadAwait.WithLabelValues(device).Set(rAwait)
		iostatWriteAwait.WithLabelValues(device).Set(wAwait)
		iostatAwait.WithLabelValues(device).Set(await)
		iostatUtilPercent.WithLabelValues(device).Set(util)
	}
}


func main() {
	// 注册 Collector
	prometheus.MustRegister(&PidstatCollector{})
	prometheus.MustRegister(&VmstatCollector{})
	prometheus.MustRegister(&IostatCollector{})

	// 暴露 /metrics 端口
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Starting node-detail-exporter on :9101")
	log.Fatal(http.ListenAndServe(":9101", nil))
}