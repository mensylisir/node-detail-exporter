package collector

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewPidstatCollector(t *testing.T) {
	c, err := NewPidstatCollector()
	if err != nil {
		t.Fatalf("NewPidstatCollector() error = %v", err)
	}
	if c == nil {
		t.Fatal("NewPidstatCollector() returned nil")
	}
}

func TestPidstatCollectorUpdate(t *testing.T) {
	// This is a basic test to ensure the collector runs without panicking.
	c, err := NewPidstatCollector()
	if err != nil {
		t.Fatalf("NewPidstatCollector() error = %v", err)
	}

	ch := make(chan prometheus.Metric)
	go func() {
		for range ch {
			// consume metrics
		}
	}()

	err = c.Update(ch, 10*time.Second)
	if err != nil {
		t.Errorf("PidstatCollector.Update() error = %v", err)
	}
	close(ch)
}
