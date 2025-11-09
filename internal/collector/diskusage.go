package collector

import (
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/disk"
)

var (
	diskUsageTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_disk_usage_total_bytes",
			Help: "Total disk space by device.",
		},
		[]string{"device", "mountpoint", "fstype"},
	)
	diskUsageUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_disk_usage_used_bytes",
			Help: "Used disk space by device.",
		},
		[]string{"device", "mountpoint", "fstype"},
	)
	diskUsageFree = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_disk_usage_free_bytes",
			Help: "Free disk space by device.",
		},
		[]string{"device", "mountpoint", "fstype"},
	)
	diskUsageUsedPercent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_disk_usage_used_percent",
			Help: "Percentage of disk space used by device.",
		},
		[]string{"device", "mountpoint", "fstype"},
	)
)

// DiskUsageCollector collects disk usage metrics.
type DiskUsageCollector struct{}

// NewDiskUsageCollector returns a new DiskUsageCollector.
func NewDiskUsageCollector() (Collector, error) {
	return &DiskUsageCollector{}, nil
}

// Update implements the Collector interface.
func (c *DiskUsageCollector) Name() string {
	return "diskusage"
}

func (c *DiskUsageCollector) Update(ch chan<- prometheus.Metric, interval time.Duration) error {
	diskUsageTotal.Reset()
	diskUsageUsed.Reset()
	diskUsageFree.Reset()
	diskUsageUsedPercent.Reset()

	partitions, err := disk.Partitions(true)
	if err != nil {
		log.Printf("Error getting disk partitions: %v", err)
		return err
	}

	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			log.Printf("Error getting disk usage for %s: %v", partition.Mountpoint, err)
			continue
		}

		labels := prometheus.Labels{
			"device":     partition.Device,
			"mountpoint": usage.Path,
			"fstype":     usage.Fstype,
		}

		diskUsageTotal.With(labels).Set(float64(usage.Total))
		diskUsageUsed.With(labels).Set(float64(usage.Used))
		diskUsageFree.With(labels).Set(float64(usage.Free))
		diskUsageUsedPercent.With(labels).Set(usage.UsedPercent)
	}

	diskUsageTotal.Collect(ch)
	diskUsageUsed.Collect(ch)
	diskUsageFree.Collect(ch)
	diskUsageUsedPercent.Collect(ch)

	return nil
}
