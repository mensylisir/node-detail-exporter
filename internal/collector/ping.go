package collector

import (
	"bufio"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	pingLatencyMin = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_ping_latency_min_seconds",
			Help: "Minimum ping latency to a target.",
		},
		[]string{"target"},
	)
	pingLatencyAvg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_ping_latency_avg_seconds",
			Help: "Average ping latency to a target.",
		},
		[]string{"target"},
	)
	pingLatencyMax = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_ping_latency_max_seconds",
			Help: "Maximum ping latency to a target.",
		},
		[]string{"target"},
	)
	pingLatencyStddev = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_ping_latency_stddev_seconds",
			Help: "Standard deviation of ping latency to a target.",
		},
		[]string{"target"},
	)
	pingPacketLoss = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_ping_packet_loss_percent",
			Help: "Packet loss percentage to a target.",
		},
		[]string{"target"},
	)
)

type PingCollector struct {
	Targets []string
}

func NewPingCollector(targets []string) (Collector, error) {
	return &PingCollector{Targets: targets}, nil
}

func (c *PingCollector) Update(ch chan<- prometheus.Metric) error {
	for _, target := range c.Targets {
		ping(target)
	}

	pingLatencyMin.Collect(ch)
	pingLatencyAvg.Collect(ch)
	pingLatencyMax.Collect(ch)
	pingLatencyStddev.Collect(ch)
	pingPacketLoss.Collect(ch)

	return nil
}

func ping(target string) {
	cmd := exec.Command("ping", "-c", "5", "-W", "1", target)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error creating stdout pipe for ping: %v", err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Error starting ping: %v", err)
		return
	}

	parsePing(stdout, target)

	cmd.Wait()
}

func parsePing(stdout io.Reader, target string) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "packet loss") {
			re := regexp.MustCompile(`(\d+)% packet loss`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				loss, _ := strconv.ParseFloat(matches[1], 64)
				pingPacketLoss.WithLabelValues(target).Set(loss)
			}
		}

		if strings.Contains(line, "rtt min/avg/max/mdev") {
			re := regexp.MustCompile(`= ([\d.]+)/([\d.]+)/([\d.]+)/([\d.]+) ms`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 4 {
				min, _ := strconv.ParseFloat(matches[1], 64)
				avg, _ := strconv.ParseFloat(matches[2], 64)
				max, _ := strconv.ParseFloat(matches[3], 64)
				stddev, _ := strconv.ParseFloat(matches[4], 64)
				pingLatencyMin.WithLabelValues(target).Set(min / 1000)
				pingLatencyAvg.WithLabelValues(target).Set(avg / 1000)
				pingLatencyMax.WithLabelValues(target).Set(max / 1000)
				pingLatencyStddev.WithLabelValues(target).Set(stddev / 1000)
			}
		}
	}
}
