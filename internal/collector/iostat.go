package collector

import (
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/disk"
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

type IostatCollector struct {
	prevIOCounters map[string]disk.IOCountersStat
	mu             sync.Mutex
}

func NewIostatCollector() (Collector, error) {
	return &IostatCollector{
		prevIOCounters: make(map[string]disk.IOCountersStat),
	}, nil
}

func (c *IostatCollector) Name() string {
	return "iostat"
}

func (c *IostatCollector) Update(ch chan<- prometheus.Metric, interval time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	iostatAwait.Reset()
	iostatReadAwait.Reset()
	iostatWriteAwait.Reset()
	iostatUtilPercent.Reset()

	ioCounters, err := disk.IOCounters()
	if err != nil {
		log.Printf("Error getting disk IO counters: %v", err)
		return err
	}

	for device, counter := range ioCounters {
		if prevCounter, ok := c.prevIOCounters[device]; ok {
			readTimeDelta := counter.ReadTime - prevCounter.ReadTime
			writeTimeDelta := counter.WriteTime - prevCounter.WriteTime
			readCountDelta := counter.ReadCount - prevCounter.ReadCount
			writeCountDelta := counter.WriteCount - prevCounter.WriteCount
			ioTimeDelta := counter.IoTime - prevCounter.IoTime

			var rAwait, wAwait, await, util float64
			if readCountDelta > 0 {
				rAwait = float64(readTimeDelta) / float64(readCountDelta)
			}
			if writeCountDelta > 0 {
				wAwait = float64(writeTimeDelta) / float64(writeCountDelta)
			}
			if readCountDelta+writeCountDelta > 0 {
				await = float64(readTimeDelta+writeTimeDelta) / float64(readCountDelta+writeCountDelta)
			}
			if interval.Milliseconds() > 0 {
				util = float64(ioTimeDelta) / float64(interval.Milliseconds()) * 100
			}

			iostatReadAwait.WithLabelValues(device).Set(rAwait)
			iostatWriteAwait.WithLabelValues(device).Set(wAwait)
			iostatAwait.WithLabelValues(device).Set(await)
			iostatUtilPercent.WithLabelValues(device).Set(util)
		}
	}

	c.prevIOCounters = ioCounters

	iostatAwait.Collect(ch)
	iostatReadAwait.Collect(ch)
	iostatWriteAwait.Collect(ch)
	iostatUtilPercent.Collect(ch)

	return nil
}
