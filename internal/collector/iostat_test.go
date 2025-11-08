package collector

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewIostatCollector(t *testing.T) {
	c, err := NewIostatCollector()
	if err != nil {
		t.Fatalf("NewIostatCollector() error = %v", err)
	}
	if c == nil {
		t.Fatal("NewIostatCollector() returned nil")
	}
}

func TestIostatCollectorUpdate(t *testing.T) {
	// This is a basic test to ensure the collector runs without panicking.
	c, err := NewIostatCollector()
	if err != nil {
		t.Fatalf("NewIostatCollector() error = %v", err)
	}

	ch := make(chan prometheus.Metric)
	go func() {
		for range ch {
			// consume metrics
		}
	}()

	err = c.Update(ch, 10*time.Second)
	if err != nil {
		t.Errorf("IostatCollector.Update() error = %v", err)
	}
	close(ch)
}
