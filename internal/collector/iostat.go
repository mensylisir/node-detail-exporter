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

type IostatCollector struct{}

func NewIostatCollector() (Collector, error) {
	return &IostatCollector{}, nil
}

func (c *IostatCollector) Update(ch chan<- prometheus.Metric) error {
	iostatAwait.Reset()
	iostatReadAwait.Reset()
	iostatWriteAwait.Reset()
	iostatUtilPercent.Reset()

	parser.CollectFromCommand("iostat", []string{"-x", "-d", "1", "1"}, parseIostat)

	iostatAwait.Collect(ch)
	iostatReadAwait.Collect(ch)
	iostatWriteAwait.Collect(ch)
	iostatUtilPercent.Collect(ch)

	return nil
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
