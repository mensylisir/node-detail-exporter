package collector

import (
	"log"
	"time"

	"github.com/go-ping/ping"
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

func (c *PingCollector) Update(ch chan<- prometheus.Metric, interval time.Duration) error {
	for _, target := range c.Targets {
		executePing(target)
	}

	pingLatencyMin.Collect(ch)
	pingLatencyAvg.Collect(ch)
	pingLatencyMax.Collect(ch)
	pingLatencyStddev.Collect(ch)
	pingPacketLoss.Collect(ch)

	return nil
}

func executePing(target string) {
	pinger, err := ping.NewPinger(target)
	if err != nil {
		log.Printf("Error creating pinger for target %s: %v", target, err)
		return
	}

	pinger.Count = 5
	pinger.Timeout = time.Second * 1
	pinger.SetPrivileged(true)

	err = pinger.Run()
	if err != nil {
		log.Printf("Error running ping for target %s: %v", target, err)
		return
	}

	stats := pinger.Statistics()
	pingPacketLoss.WithLabelValues(target).Set(stats.PacketLoss)
	pingLatencyMin.WithLabelValues(target).Set(stats.MinRtt.Seconds())
	pingLatencyAvg.WithLabelValues(target).Set(stats.AvgRtt.Seconds())
	pingLatencyMax.WithLabelValues(target).Set(stats.MaxRtt.Seconds())
	pingLatencyStddev.WithLabelValues(target).Set(stats.StdDevRtt.Seconds())
}
