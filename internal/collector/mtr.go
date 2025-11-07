package collector

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os/exec"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	mtrHopLoss = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_mtr_hop_packet_loss_percent",
			Help: "Packet loss percentage for a specific hop in an MTR trace.",
		},
		[]string{"target", "hop", "host"},
	)
	mtrHopLatencyAvg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_mtr_hop_latency_avg_seconds",
			Help: "Average latency for a specific hop in an MTR trace.",
		},
		[]string{"target", "hop", "host"},
	)
	mtrHopLatencyBest = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_mtr_hop_latency_best_seconds",
			Help: "Best (minimum) latency for a specific hop in an MTR trace.",
		},
		[]string{"target", "hop", "host"},
	)
	mtrHopLatencyWorst = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_mtr_hop_latency_worst_seconds",
			Help: "Worst (maximum) latency for a specific hop in an MTR trace.",
		},
		[]string{"target", "hop", "host"},
	)
	mtrHopLatencyStdDev = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_mtr_hop_latency_stddev_seconds",
			Help: "Standard deviation of latency for a specific hop in an MTR trace.",
		},
		[]string{"target", "hop", "host"},
	)
)

// MtrCollector collects metrics from mtr.
type MtrCollector struct {
	Targets []string
}

// Structs to unmarshal mtr JSON output
type MtrReport struct {
	Report struct {
		Hubs []MtrHub `json:"hubs"`
	} `json:"report"`
}

type MtrHub struct {
	Count int     `json:"count"`
	Host  string  `json:"host"`
	Loss  float64 `json:"Loss%"`
	Avg   float64 `json:"Avg"`
	Best  float64 `json:"Best"`
	Wrst  float64 `json:"Wrst"`
	StDev float64 `json:"StDev"`
}

// NewMtrCollector returns a new MtrCollector.
func NewMtrCollector(targets []string) (Collector, error) {
	return &MtrCollector{Targets: targets}, nil
}

// Update implements the Collector interface.
func (c *MtrCollector) Update(ch chan<- prometheus.Metric) error {
	mtrHopLoss.Reset()
	mtrHopLatencyAvg.Reset()
	mtrHopLatencyBest.Reset()
	mtrHopLatencyWorst.Reset()
	mtrHopLatencyStdDev.Reset()

	for _, target := range c.Targets {
		err := c.collectMtrMetrics(target)
		if err != nil {
			log.Printf("Error collecting mtr metrics for target %s: %v", target, err)
		}
	}

	mtrHopLoss.Collect(ch)
	mtrHopLatencyAvg.Collect(ch)
	mtrHopLatencyBest.Collect(ch)
	mtrHopLatencyWorst.Collect(ch)
	mtrHopLatencyStdDev.Collect(ch)

	return nil
}

func (c *MtrCollector) collectMtrMetrics(target string) error {
	// -j for json output, -c 5 for 5 packets
	cmd := exec.Command("mtr", "-j", "-c", "5", target)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	data, err := ioutil.ReadAll(stdout)
	if err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	var report MtrReport
	if err := json.Unmarshal(data, &report); err != nil {
		log.Printf("Error parsing mtr JSON output for target %s: %v", target, err)
		return err
	}

	for _, hub := range report.Report.Hubs {
		hop := strconv.Itoa(hub.Count)
		// Latencies from mtr are in milliseconds, convert to seconds for prometheus
		mtrHopLoss.WithLabelValues(target, hop, hub.Host).Set(hub.Loss)
		mtrHopLatencyAvg.WithLabelValues(target, hop, hub.Host).Set(hub.Avg / 1000)
		mtrHopLatencyBest.WithLabelValues(target, hop, hub.Host).Set(hub.Best / 1000)
		mtrHopLatencyWorst.WithLabelValues(target, hop, hub.Host).Set(hub.Wrst / 1000)
		mtrHopLatencyStdDev.WithLabelValues(target, hop, hub.Host).Set(hub.StDev / 1000)
	}
	return nil
}
