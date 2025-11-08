package collector

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewDiskUsageCollector(t *testing.T) {
	c, err := NewDiskUsageCollector()
	if err != nil {
		t.Fatalf("NewDiskUsageCollector() error = %v", err)
	}
	if c == nil {
		t.Fatal("NewDiskUsageCollector() returned nil")
	}
}

func TestDiskUsageCollectorUpdate(t *testing.T) {
	// This is a basic test to ensure the collector runs without panicking.
	c, err := NewDiskUsageCollector()
	if err != nil {
		t.Fatalf("NewDiskUsageCollector() error = %v", err)
	}

	ch := make(chan prometheus.Metric)
	go func() {
		for range ch {
			// consume metrics
		}
	}()

	err = c.Update(ch, 10*time.Second)
	if err != nil {
		t.Errorf("DiskUsageCollector.Update() error = %v", err)
	}
	close(ch)
}
